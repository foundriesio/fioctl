package keys

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	canonical "github.com/docker/go/canonical/json"
	"github.com/spf13/viper"
	tuf "github.com/theupdateframework/notary/tuf/data"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands"
)

var errFoundNoKey = errors.New("Found no active signing key")

type OfflineCreds map[string][]byte

type TufSigner struct {
	Id   string
	Type TufKeyType
	Key  crypto.Signer
}

type TufKeyPair struct {
	signer       TufSigner
	atsPriv      client.AtsKey
	atsPrivBytes []byte
	atsPub       client.AtsKey
	atsPubBytes  []byte
}

func ParseTufKeyType(s string) TufKeyType {
	t, err := parseTufKeyType(s)
	subcommands.DieNotNil(err)
	return t
}

func ParseTufRoleNameOffline(s string) string {
	r, err := parseTufRoleName(s, tufRoleNameRoot, tufRoleNameTargets)
	subcommands.DieNotNil(err)
	return r
}

func ParseTufRoleNameOnline(s string) string {
	r, err := parseTufRoleName(s, tufRoleNameTargets, tufRoleNameSnapshot, tufRoleNameTimestamp)
	subcommands.DieNotNil(err)
	return r
}

func genTufKeyId(key crypto.Signer) string {
	// # This has to match the exact logic used by ota-tuf (required by garage-sign):
	// https://github.com/foundriesio/ota-tuf/blob/fio-changes/libtuf/src/main/scala/com/advancedtelematic/libtuf/crypt/TufCrypto.scala#L66-L71
	// It sets a keyid to a signature of the key's canonical DER encoding (same logic for all keys).
	// Note: this differs from the TUF spec, need to change once we deprecate the garage-sign.
	pubBytes, err := x509.MarshalPKIXPublicKey(key.Public())
	subcommands.DieNotNil(err)
	return fmt.Sprintf("%x", sha256.Sum256(pubBytes))
}

func genTufKeyPair(keyType TufKeyType) TufKeyPair {
	keyTypeName := keyType.Name()
	pk, err := keyType.GenerateKey()
	subcommands.DieNotNil(err)
	privKey, pubKey, err := keyType.SaveKeyPair(pk)
	subcommands.DieNotNil(err)

	priv := client.AtsKey{
		KeyType:  keyTypeName,
		KeyValue: client.AtsKeyVal{Private: privKey},
	}
	atsPrivBytes, err := json.Marshal(priv)
	subcommands.DieNotNil(err)

	pub := client.AtsKey{
		KeyType:  keyTypeName,
		KeyValue: client.AtsKeyVal{Public: pubKey},
	}
	atsPubBytes, err := json.Marshal(pub)
	subcommands.DieNotNil(err)

	id := genTufKeyId(pk)

	return TufKeyPair{
		atsPriv:      priv,
		atsPrivBytes: atsPrivBytes,
		atsPub:       pub,
		atsPubBytes:  atsPubBytes,
		signer: TufSigner{
			Id:   id,
			Type: keyType,
			Key:  pk,
		},
	}
}

func SignTufMeta(metaBytes []byte, signers ...TufSigner) ([]tuf.Signature, error) {
	signatures := make([]tuf.Signature, len(signers))

	for idx, signer := range signers {
		digest := metaBytes[:]
		opts := signer.Type.SigOpts()
		if opts.HashFunc() != crypto.Hash(0) {
			// Golang expects the caller to hash the digest if needed by the signing method

			h := opts.HashFunc().New()
			h.Write(digest)
			digest = h.Sum(nil)
		}
		sigBytes, err := signer.Key.Sign(rand.Reader, digest, opts)
		if err != nil {
			return nil, err
		}
		signatures[idx] = tuf.Signature{
			KeyID:     signer.Id,
			Method:    tuf.SigAlgorithm(signer.Type.SigName()),
			Signature: sigBytes,
		}
	}
	return signatures, nil
}

func signTufRoot(root *client.AtsTufRoot, signers ...TufSigner) error {
	bytes, err := canonical.MarshalCanonical(root.Signed)
	if err != nil {
		return err
	}
	signatures, err := SignTufMeta(bytes, signers...)
	if err != nil {
		return err
	}
	for _, oldSig := range root.Signatures {
		found := false
		for _, newSig := range signatures {
			if oldSig.KeyID == newSig.KeyID {
				found = true
				break
			}
		}
		if !found {
			signatures = append(signatures, oldSig)
		}
	}
	root.Signatures = signatures
	return nil
}

func saveTufCreds(path string, creds OfflineCreds) {
	file, err := os.Create(path)
	subcommands.DieNotNil(err)
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for name, val := range creds {
		header := &tar.Header{
			Name: name,
			Size: int64(len(val)),
		}
		subcommands.DieNotNil(tarWriter.WriteHeader(header))
		_, err := tarWriter.Write(val)
		subcommands.DieNotNil(err)
	}
}

func saveTempTufCreds(credsFile string, creds OfflineCreds) string {
	path := credsFile + ".tmp"
	if _, err := os.Stat(path); err == nil {
		subcommands.DieNotNil(fmt.Errorf(`Backup file exists: %s
This file may be from a previous failed key rotation and include critical data.
Please move this file somewhere safe before re-running this command.`,
			path,
		))
	}
	saveTufCreds(path, creds)
	return path
}

func GetOfflineCreds(credsFile string) (OfflineCreds, error) {
	f, err := os.Open(credsFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	files := make(OfflineCreds)

	gzf, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gzf)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		} else if err != nil {
			return nil, err
		}

		if hdr.Typeflag == tar.TypeDir {
			continue
		}

		var b bytes.Buffer
		if _, err = io.Copy(&b, tr); err != nil {
			return nil, err
		}
		files[hdr.Name] = b.Bytes()
	}
	return files, nil
}

func FindOneTufSigner(root *client.AtsTufRoot, creds OfflineCreds, keyids []string) (signer TufSigner, err error) {
	var signers []TufSigner
	if signers, err = findTufSigners(root, creds, keyids); err == nil {
		if len(signers) == 0 {
			err = fmt.Errorf("%w for: %v.", errFoundNoKey, keyids)
		} else if len(signers) > 1 {
			err = fmt.Errorf(`Found more than one active signing key for: %v.
This is an unsupported and insecure way to store private keys.
Please, provide a keys file which contains a single active signing key.`, keyids)
		} else {
			signer = signers[0]
		}
	}
	return
}

func checkNoTufSigner(root *client.AtsTufRoot, creds OfflineCreds, keyids []string) (err error) {
	var signers []TufSigner
	if signers, err = findTufSigners(root, creds, keyids); err == nil {
		if len(signers) > 0 {
			err = errors.New("It is not allowed to store more than one active signing key into one file")
		}
	}
	return
}

func findTufSigners(root *client.AtsTufRoot, creds OfflineCreds, keyids []string) ([]TufSigner, error) {
	// Look in creds for each candidate from keyids and return all private keys that match
	matchPubKeys := make(map[string]client.AtsKey, len(keyids))
	for _, kid := range keyids {
		if pk, ok := root.Signed.Keys[kid]; ok {
			pk.KeyValue.Public = strings.TrimSpace(pk.KeyValue.Public)
			matchPubKeys[kid] = pk
		} else {
			return nil, fmt.Errorf("Unable to find key %s in root.json", kid)
		}
	}

	matchSigners := make([]TufSigner, 0, 1) // Normally, we find none or one match
	for file, bytes := range creds {
		if !strings.HasSuffix(file, ".pub") {
			continue
		}

		var key client.AtsKey
		if err := json.Unmarshal(bytes, &key); err != nil {
			return nil, fmt.Errorf("Unable to parse JSON for %s: %w", file, err)
		}
		probe := strings.TrimSpace(key.KeyValue.Public)

		var matchId, matchKeyType string
		for kid, match := range matchPubKeys {
			if match.KeyValue.Public == probe {
				matchId = kid
				matchKeyType = match.KeyType
				break
			}
		}
		if len(matchId) == 0 {
			continue
		}

		file = strings.Replace(file, ".pub", ".sec", 1)
		bytes = creds[file]
		if err := json.Unmarshal(bytes, &key); err != nil {
			return nil, fmt.Errorf("Unable to parse JSON for %s: %w", file, err)
		}
		if key.KeyType != matchKeyType {
			return nil, fmt.Errorf("Mismatch in key type for %s: %s != %s", file, key.KeyType, matchKeyType)
		}
		keyType, err := parseTufKeyType(key.KeyType)
		if err != nil {
			return nil, fmt.Errorf("Unsupported key type for %s: %s", file, key.KeyType)
		}
		signer, err := keyType.ParseKey(key.KeyValue.Private)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse key value for %s: %w", file, err)
		}
		matchSigners = append(matchSigners, TufSigner{
			Id:   matchId,
			Type: keyType,
			Key:  signer,
		})
	}
	return matchSigners, nil
}

func addOfflineTufKey(
	root *client.AtsTufRoot, role tuf.RoleName, key TufKeyPair, oldKids []string, creds OfflineCreds,
) {
	base := fmt.Sprintf("tufrepo/keys/fioctl-%s-%s", role, key.signer.Id)
	creds[base+".pub"] = key.atsPubBytes
	creds[base+".sec"] = key.atsPrivBytes
	root.Signed.Keys[key.signer.Id] = key.atsPub
	root.Signed.Roles[role].KeyIDs = append(oldKids, key.signer.Id)

	factory := viper.GetString("factory")
	user, err := api.UserAccessDetails(factory, "self")
	subcommands.DieNotNil(err)
	if root.Signed.KeyOwners == nil {
		root.Signed.KeyOwners = make(map[string]client.RootKeyOwner)
	}
	root.Signed.KeyOwners[key.signer.Id] = client.RootKeyOwner{
		PolisId:   user.PolisId,
		CreatedAt: time.Unix(time.Now().Unix(), 0).UTC(), // Strip millis
	}
}

func removeUnusedTufKeys(root *client.AtsTufRoot) {
	var inuse []string
	for _, role := range root.Signed.Roles {
		inuse = append(inuse, role.KeyIDs...)
	}

	for k := range root.Signed.Keys {
		// is k in inuse?
		found := false
		for _, val := range inuse {
			if k == val {
				found = true
				break
			}
		}
		if !found {
			fmt.Println("= Removing unused key:", k)
			delete(root.Signed.Keys, k)
		}
	}
}

func checkTufRootUpdatesStatus(updates client.TufRootUpdates, forUpdate bool) (
	curCiRoot, newCiRoot, newProdRoot *client.AtsTufRoot,
) {
	switch updates.Status {
	case client.TufRootUpdatesStatusNone:
		if forUpdate {
			subcommands.DieNotNil(errors.New(`There are no TUF root updates in progress.
Please, run 'fioctl keys tuf updates init' to start over.`))
		}
	case client.TufRootUpdatesStatusStarted:
		break
	case client.TufRootUpdatesStatusApplying:
		if forUpdate {
			subcommands.DieNotNil(errors.New(
				"No modifications to TUF root updates allowed while they are being applied.",
			))
		}
	default:
		subcommands.DieNotNil(fmt.Errorf("Unexpected TUF root updates status: %s", updates.Status))
	}

	if updates.Current != nil && updates.Current.CiRoot != "" {
		subcommands.DieNotNil(
			json.Unmarshal([]byte(updates.Current.CiRoot), &curCiRoot), "Current CI root",
		)
	}
	if curCiRoot == nil {
		subcommands.DieNotNil(errors.New("Current TUF CI root not set. Please, report a bug."))
	}
	if updates.Updated != nil {
		if updates.Updated.CiRoot != "" {
			subcommands.DieNotNil(
				json.Unmarshal([]byte(updates.Updated.CiRoot), &newCiRoot), "Updated CI root",
			)
		}
		if updates.Updated.ProdRoot != "" {
			subcommands.DieNotNil(
				json.Unmarshal([]byte(updates.Updated.ProdRoot), &newProdRoot), "Updated prod root",
			)
		}
	}
	if newCiRoot == nil && updates.Status != client.TufRootUpdatesStatusNone {
		subcommands.DieNotNil(errors.New("Updated TUF CI root not set. Please, report a bug."))
	}
	return
}

func genProdTufRoot(ciRoot *client.AtsTufRoot) (prodRoot *client.AtsTufRoot) {
	// Deep copy in Golang is hard; use the marshal-unmarshal trick
	body, err := json.Marshal(ciRoot)
	subcommands.DieNotNil(err)
	subcommands.DieNotNil(json.Unmarshal(body, &prodRoot))
	prodRoot.Signed.Roles["targets"].Threshold = 2
	return
}

func finalizeTufRootChanges(ciRoot, prodRoot *client.AtsTufRoot) (newCiRoot, newProdRoot *client.AtsTufRoot) {
	// This function must be called after any changes to the TUF root signed body
	newCiRoot = ciRoot
	newCiRoot.Signatures = make([]tuf.Signature, 0)
	removeUnusedTufKeys(newCiRoot)
	newProdRoot = genProdTufRoot(newCiRoot)
	if prodRoot != nil {
		newProdRoot.Signed.Roles["targets"].Threshold = prodRoot.Signed.Roles["targets"].Threshold
	}
	return
}

func signNewTufRoot(curCiRoot, newCiRoot, newProdRoot *client.AtsTufRoot, creds OfflineCreds) {
	// Find new and old keys that match; at least one of them must be found.
	signers := make([]TufSigner, 0, 2)
	newKey, newErr := FindOneTufSigner(newCiRoot, creds, newCiRoot.Signed.Roles["root"].KeyIDs)
	if !errors.Is(newErr, errFoundNoKey) {
		subcommands.DieNotNil(newErr)
		signers = append(signers, newKey)
	}
	oldKey, oldErr := FindOneTufSigner(curCiRoot, creds, curCiRoot.Signed.Roles["root"].KeyIDs)
	if !errors.Is(oldErr, errFoundNoKey) {
		subcommands.DieNotNil(oldErr)
		if len(signers) == 0 || oldKey.Id != newKey.Id {
			signers = append(signers, oldKey)
		}
	}

	// At this point either oldKey or newKey was found, or both newErr and oldErr are errFoundNoKey
	if len(signers) == 0 {
		subcommands.DieNotNil(fmt.Errorf("%s\n%s", oldErr, newErr))
	}

	fmt.Println("= Signing new TUF root")
	subcommands.DieNotNil(signTufRoot(newCiRoot, signers...))
	subcommands.DieNotNil(signTufRoot(newProdRoot, signers...))
}

func signProdTargets(
	factory string, signer TufSigner, excludeFunc func(string, client.AtsTufTargets) bool,
) (map[string][]tuf.Signature, error) {
	targetsMap, err := api.ProdTargetsList(factory, false)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch production targets: %w", err)
	} else if targetsMap == nil {
		return nil, nil
	}

	signatureMap := make(map[string][]tuf.Signature)
	for tag, targets := range targetsMap {
		if excludeFunc != nil && excludeFunc(tag, targets) {
			continue
		}
		bytes, err := canonical.MarshalCanonical(targets.Signed)
		if err != nil {
			return nil, fmt.Errorf("Failed to marshal targets for tag %s: %w", tag, err)
		}
		signatures, err := SignTufMeta(bytes, signer)
		if err != nil {
			return nil, fmt.Errorf("Failed to re-sign targets for tag %s: %w", tag, err)
		}
		signatureMap[tag] = signatures
	}
	return signatureMap, nil
}
