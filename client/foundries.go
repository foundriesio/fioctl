package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	tuf "github.com/theupdateframework/notary/tuf/data"
)

type Api struct {
	serverUrl string
	apiToken  string
}

type Device struct {
	Uuid       string `json:"uuid"`
	Name       string `json:"name"`
	Owner      string `json:"owner"`
	Factory    string `json:"factory"`
	CreatedAt  string `json:"created-at"`
	LastSeen   string `json:"last-seen"`
	OstreeHash string `json:"ostree-hash"`
}

type DeviceList struct {
	Devices []Device `json:"devices"`
	Total   int      `json:"total"`
	Next    *string  `json:"next"`
}

type TufCustom struct {
	HardwareIds  []string `json:"hardwareIds"`
	TargetFormat string  `json:"targetFormat"`
}

func NewApiClient(serverUrl, apiToken string) *Api {
	api := Api{strings.TrimRight(serverUrl, "/"), apiToken}
	return &api
}

func (a *Api) Get(url string) (*[]byte, error) {
	client := http.Client{
		Timeout: time.Second * 2,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "fioctl")
	req.Header.Set("OSF-TOKEN", a.apiToken)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Unable to get '%s': HTTP_%d\n=%s", url, res.StatusCode, body)
	}
	return &body, nil
}

func (a *Api) DeviceList() (*DeviceList, error) {
	return a.DeviceListCont(a.serverUrl + "/ota/devices/")
}

func (a *Api) DeviceListCont(url string) (*DeviceList, error) {
	body, err := a.Get(url)
	if err != nil {
		return nil, err
	}

	devices := DeviceList{}
	err = json.Unmarshal(*body, &devices)
	if err != nil {
		return nil, err
	}
	return &devices, nil
}

func (a *Api) TargetsListRaw(factory string) (*[]byte, error) {
	url := a.serverUrl + "/ota/repo/" + factory + "/api/v1/user_repo/targets.json"
	return a.Get(url)
}

func (a *Api) TargetsList(factory string) (*tuf.SignedTargets, error) {
	body, err := a.TargetsListRaw(factory)
	if err != nil {
		return nil, err
	}
	targets := tuf.SignedTargets{}
	err = json.Unmarshal(*body, &targets)
	if err != nil {
		return nil, err
	}

	return &targets, nil
}

func (a *Api) TargetCustom(target tuf.FileMeta) (*TufCustom, error) {
	custom := TufCustom{}
	err := json.Unmarshal(*target.Custom, &custom)
	if err != nil {
		return nil, err
	}
	return &custom, nil
}
