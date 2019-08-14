package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
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

func NewApiClient(serverUrl, apiToken string) *Api {
	api := Api{strings.TrimRight(serverUrl, "/"), apiToken}
	return &api
}

func (a *Api) DeviceList() (*DeviceList, error) {
	return a.DeviceListCont(a.serverUrl + "/ota/devices/")
}

func (a *Api) DeviceListCont(url string) (*DeviceList, error) {
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

	devices := DeviceList{}
	err = json.Unmarshal(body, &devices)
	if err != nil {
		return nil, err
	}
	return &devices, nil
}
