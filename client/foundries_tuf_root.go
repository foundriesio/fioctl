package client

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	tuf "github.com/theupdateframework/notary/tuf/data"
)

// ota-tuf serializes root.json differently from Notary. The key representation
// and signing algoritms differ slightly. These Ats* structs provide an
// an implementation compatible with ota-tuf and libaktualizr.
type AtsKeyVal struct {
	Public  string `json:"public,omitempty"`
	Private string `json:"private,omitempty"`
}
type AtsKey struct {
	KeyType  string    `json:"keytype"`
	KeyValue AtsKeyVal `json:"keyval"`
}

type RootChangeReason struct {
	PolisId   string    `json:"polis-id"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}
type AtsRootMeta struct {
	tuf.SignedCommon
	Consistent bool                           `json:"consistent_snapshot"`
	Keys       map[string]AtsKey              `json:"keys"`
	Roles      map[tuf.RoleName]*tuf.RootRole `json:"roles"`
	Reason     *RootChangeReason              `json:"x-changelog,omitempty"`
}

type AtsTufRoot struct {
	// A non-standard targets-signatures field allows to make an atomic key rotation
	TargetsSignatures map[string][]tuf.Signature `json:"targets-signatures,omitempty"`
	Signatures        []tuf.Signature            `json:"signatures"`
	Signed            AtsRootMeta                `json:"signed"`
}

type TufRootUpdatesInit struct {
	TransactionId string `json:"txid"`
}

func (a *Api) TufTargetsOnlineKey(factory string) (*AtsKey, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/ci-targets.pub"
	body, err := a.Get(url)
	if err != nil {
		return nil, err
	}
	key := AtsKey{}
	err = json.Unmarshal(*body, &key)
	return &key, err
}

func (a *Api) TufRootFirstKey(factory string) (*AtsKey, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/first_root.sec"
	body, err := a.Get(url)
	if err != nil {
		return nil, err
	}
	root := AtsKey{}
	err = json.Unmarshal(*body, &root)
	return &root, err
}

func (a *Api) TufRootGet(factory string) (*AtsTufRoot, error) {
	return a.tufRootGet(factory, false, -1)
}

func (a *Api) TufRootGetVer(factory string, version int) (*AtsTufRoot, error) {
	return a.tufRootGet(factory, false, version)
}

func (a *Api) TufProdRootGet(factory string) (*AtsTufRoot, error) {
	return a.tufRootGet(factory, true, -1)
}

func (a *Api) TufRootPost(factory string, root []byte) (string, error) {
	return a.tufRootPost(factory, false, root)
}

func (a *Api) TufProdRootPost(factory string, root []byte) (string, error) {
	return a.tufRootPost(factory, true, root)
}

func (a *Api) TufRootUpdatesInit(factory, changelog string) (res TufRootUpdatesInit, err error) {
	var body *[]byte
	url := a.serverUrl + "/ota/repo/" + factory + "/api/v1/user_repo/root/updates"
	data, _ := json.Marshal(map[string]interface{}{"message": "changelog"})
	if body, err = a.Post(url, data); err == nil {
		err = json.Unmarshal(*body, &res)
	} else if herr := AsHttpError(err); herr != nil && herr.Response.StatusCode == 409 {
		herr.Message += "\n=Only one TUF root updates transaction can be active at a time"
	}
	return
}

func (a *Api) tufRootGet(factory string, prod bool, version int) (*AtsTufRoot, error) {
	url := a.serverUrl + "/ota/repo/" + factory + "/api/v1/user_repo/"
	if version > 0 {
		url += fmt.Sprintf("%d.", version)
	}
	url += "root.json"
	if prod {
		url += "?production=1"
	}
	logrus.Debugf("Fetch root %s", url)
	body, err := a.Get(url)
	if err != nil {
		return nil, err
	}
	root := AtsTufRoot{}
	err = json.Unmarshal(*body, &root)
	return &root, err
}

func (a *Api) tufRootPost(factory string, prod bool, root []byte) (string, error) {
	url := a.serverUrl + "/ota/repo/" + factory + "/api/v1/user_repo/root"
	if prod {
		url += "?production=1"
	}
	body, err := a.Post(url, root)
	if body != nil {
		return string(*body), err
	}
	return "", err
}
