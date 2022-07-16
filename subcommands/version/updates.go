package version

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/hashicorp/go-version"
	tuf "github.com/theupdateframework/go-tuf/client"
	"github.com/theupdateframework/go-tuf/data"
)

type FioctlUpdateFinder struct {
	tuf        *tuf.Client
	localStore *jsonFilesStore
	platform   string
}

type FioctlUpdate struct {
	Version *version.Version
	Uri     string
	Sha512  data.HexBytes
	len     int64
}

func NewFioctlUpdateFinder(localJsonPath string) (*FioctlUpdateFinder, error) {
	remote, err := tuf.HTTPRemoteStore("https://raw.githubusercontent.com/doanac/fioctl/tuf/repository", nil, nil)
	if err != nil {
		return nil, err
	}
	local := &jsonFilesStore{localJsonPath}
	tuf := tuf.NewClient(local, remote)

	platform := runtime.GOOS + "/" + runtime.GOARCH
	return &FioctlUpdateFinder{tuf, local, platform}, nil
}

type fioctlCustom struct {
	Platform string `json:"platform"`
	Uri      string `json:"uri"`
	Version  string `json:"version"`
}

func (f *FioctlUpdateFinder) FindLatest() (*FioctlUpdate, error) {
	latestVer, err := version.NewVersion(Commit)
	if err != nil {
		return nil, fmt.Errorf("unable to parse binary's version: %s", err)
	}

	if !f.localStore.initialized() {
		fmt.Println("Initializing local TUF storage")
		if err := f.tuf.Init([]byte(inititalRoot)); err != nil {
			return nil, err
		}
	}
	if _, err = f.tuf.Update(); err != nil {
		return nil, err
	}
	targets, err := f.tuf.Targets()
	if err != nil {
		return nil, err
	}

	var update *FioctlUpdate
	for name, target := range targets {
		var custom fioctlCustom
		if err := json.Unmarshal(*target.Custom, &custom); err != nil {
			return nil, fmt.Errorf("unable to parse TUF data for %s: %s", name, err)
		}
		if target.Length == 0 {
			return nil, fmt.Errorf("target %s has invalid length: %d", name, target.Length)
		}
		if custom.Platform == f.platform {
			v, err := version.NewVersion(custom.Version)
			if err != nil {
				return nil, fmt.Errorf("unable to parse TUF data version for %s: %s", name, err)
			}
			if v.Core().GreaterThan(latestVer.Core()) {
				update = &FioctlUpdate{
					Sha512:  target.Hashes["sha512"],
					Version: v,
					Uri:     custom.Uri,
					len:     target.Length,
				}
				latestVer = v
			}
		}
	}
	return update, nil
}

type jsonFilesStore struct {
	path string
}

func (s jsonFilesStore) GetMeta() (map[string]json.RawMessage, error) {
	bytes, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make(map[string]json.RawMessage), nil
		}
		return nil, err
	}
	var files map[string]json.RawMessage
	if err := json.Unmarshal(bytes, &files); err != nil {
		return nil, err
	}
	return files, nil
}

func (s jsonFilesStore) SetMeta(name string, meta json.RawMessage) error {
	files, err := s.GetMeta()
	if err != nil {
		return err
	}
	if meta == nil {
		delete(files, name)
	} else {
		files[name] = meta
	}
	bytes, err := json.Marshal(files)
	if err != nil {
		return err
	}
	dst := s.path + ".tmp"
	if err = os.WriteFile(dst, bytes, 0o644); err != nil {
		return err
	}
	return os.Rename(dst, s.path)
}

func (s jsonFilesStore) DeleteMeta(name string) error {
	return s.SetMeta(name, nil)
}

func (j jsonFilesStore) Close() error {
	return nil
}

func (j jsonFilesStore) initialized() bool {
	_, err := os.Stat(j.path)
	return err == nil
}

const inititalRoot = `{"signed":{"_type":"root","consistent_snapshot":true,"expires":"2023-07-01T19:12:16Z","keys":{"00c989970de180c8c2134d2030f1594a120bc7a1664135c37721990e2c6c88e1":{"keyid_hash_algorithms":["sha256","sha512"],"keytype":"ed25519","keyval":{"public":"a168d7529b2802776962779ccf5076a26e3b681eb435c193bbdd98d6476d727a"},"scheme":"ed25519"},"3b3a6f5f0170a4664b166569b87f8fdb4151df5de1690ec5b3c542229ec39e41":{"keyid_hash_algorithms":["sha256","sha512"],"keytype":"ed25519","keyval":{"public":"2a77604025867ca4da715121997f29185018c0f29bdd018246f2a1fffe557c06"},"scheme":"ed25519"},"3f1c3c9eb25c180f500376f1a74d126902411f2004b3a524aeedc1757d00aa20":{"keyid_hash_algorithms":["sha256","sha512"],"keytype":"ed25519","keyval":{"public":"b2a5e8ea6597cafda8155031f18a40bc274aa152b0a6564ccf80d1028ef15595"},"scheme":"ed25519"},"712750a4e6d7c8715b0aa4f0eed7a9a55a5ad144066a1641829c03dcbef33e7c":{"keyid_hash_algorithms":["sha256","sha512"],"keytype":"ed25519","keyval":{"public":"c3ff3b236e4a92359523c9b7b4992a8dabd0c53a495697062e8b737245409e4d"},"scheme":"ed25519"}},"roles":{"root":{"keyids":["00c989970de180c8c2134d2030f1594a120bc7a1664135c37721990e2c6c88e1"],"threshold":1},"snapshot":{"keyids":["712750a4e6d7c8715b0aa4f0eed7a9a55a5ad144066a1641829c03dcbef33e7c"],"threshold":1},"targets":{"keyids":["3b3a6f5f0170a4664b166569b87f8fdb4151df5de1690ec5b3c542229ec39e41"],"threshold":1},"timestamp":{"keyids":["3f1c3c9eb25c180f500376f1a74d126902411f2004b3a524aeedc1757d00aa20"],"threshold":1}},"spec_version":"1.0","version":1},"signatures":[{"keyid":"00c989970de180c8c2134d2030f1594a120bc7a1664135c37721990e2c6c88e1","sig":"1373827518a965cfdb27ae6eccd35796cb24dfe6ac217316f40f48f8f6101527f993d65406303faedad38f91da3dd3184f4b14140e52e6244bdeec1e64f82206"}]}`
