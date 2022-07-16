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
	remote, err := tuf.HTTPRemoteStore(TufRepoUrl, nil, nil)
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
		if err := f.tuf.Init(InitialTufRoot); err != nil {
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

// Required to implement client.LocalStore
func (s jsonFilesStore) DeleteMeta(name string) error {
	return s.SetMeta(name, nil)
}

// Required to implement client.LocalStore
func (j jsonFilesStore) Close() error {
	return nil
}

func (j jsonFilesStore) initialized() bool {
	_, err := os.Stat(j.path)
	return err == nil
}
