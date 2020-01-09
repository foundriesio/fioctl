package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	tuf "github.com/theupdateframework/notary/tuf/data"
)

type Config struct {
	Factory           string
	Token             string
	ClientCredentials OAuthConfig
}

type Api struct {
	serverUrl string
	config    Config
}

type NetInfo struct {
	Hostname string `json:"hostname"`
	Ipv4     string `json:"local_ipv4"`
	MAC      string `json:"mac"`
}

type Update struct {
	CorrelationId string `json:"correlation-id"`
	Target        string `json:"target"`
	Version       string `json:"version"`
	Time          string `json:"time"`
}

type UpdateList struct {
	Updates []Update `json:"updates"`
	Total   int      `json:"total"`
	Next    *string  `json:"next"`
}

type EventType struct {
	Id string `json:"id"`
}

type EventDetail struct {
	Version    string `json:"version"`
	TargetName string `json:"targetName"`
	Success    *bool  `json:"success,omitempty"`
}

type UpdateEvent struct {
	Time   string      `json:"deviceTime"`
	Type   EventType   `json:"eventType"`
	Detail EventDetail `json:"event"`
}

type Device struct {
	Uuid          string           `json:"uuid"`
	Name          string           `json:"name"`
	Owner         string           `json:"owner"`
	Factory       string           `json:"factory"`
	CreatedAt     string           `json:"created-at"`
	LastSeen      string           `json:"last-seen"`
	OstreeHash    string           `json:"ostree-hash"`
	DockerApps    []string         `json:"docker-apps,omitempty"`
	Tags          []string         `json:"device-tags,omitempty"`
	Network       *NetInfo         `json:"network-info,omitempty"`
	Hardware      *json.RawMessage `json:"hardware-info,omitempty"`
	TargetName    string           `json:"target-name"`
	Status        string           `json:"status"`
	CurrentUpdate string           `json:"current-update"`
}

type DeviceList struct {
	Devices []Device `json:"devices"`
	Total   int      `json:"total"`
	Next    *string  `json:"next"`
}

type DockerApp struct {
	FileName string `json:"filename"`
	Uri      string `json:"uri"`
}

type TufCustom struct {
	HardwareIds  []string             `json:"hardwareIds,omitempty"`
	Tags         []string             `json:"tags,omitempty"`
	TargetFormat string               `json:"targetFormat,omitempty"`
	Version      string               `json:"version,omitempty"`
	DockerApps   map[string]DockerApp `json:"docker_apps,omitempty"`
}

func NewApiClient(serverUrl string, config Config) *Api {
	api := Api{strings.TrimRight(serverUrl, "/"), config}
	return &api
}

func (a *Api) RawGet(url string, headers *map[string]string) (*http.Response, error) {
	client := http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "fioctl")
	req.Header.Set("OSF-TOKEN", a.config.Token)
	if headers != nil {
		for key, val := range *headers {
			req.Header.Set(key, val)
		}
	}

	return client.Do(req)
}

func (a *Api) Get(url string) (*[]byte, error) {
	res, err := a.RawGet(url, nil)
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

func (a *Api) Patch(url string, data []byte) (*[]byte, error) {
	client := http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "fioctl")
	req.Header.Set("OSF-TOKEN", a.config.Token)
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 202 {
		return nil, fmt.Errorf("Unable to PATCH '%s': HTTP_%d\n=%s", url, res.StatusCode, body)
	}
	return &body, nil
}

func (a *Api) Delete(url string, data []byte) (*[]byte, error) {
	client := http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "fioctl")
	req.Header.Set("OSF-TOKEN", a.config.Token)
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 202 && res.StatusCode != 200 {
		return nil, fmt.Errorf("Unable to DELETE '%s': HTTP_%d\n=%s", url, res.StatusCode, body)
	}
	return &body, nil
}

func (a *Api) DeviceGet(device string) (*Device, error) {
	body, err := a.Get(a.serverUrl + "/ota/devices/" + device + "/")
	if err != nil {
		return nil, err
	}
	d := Device{}
	err = json.Unmarshal(*body, &d)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (a *Api) DeviceList(shared bool, matchTag, byFactory, nameIlike string) (*DeviceList, error) {
	sharedInt := 0
	if shared {
		sharedInt = 1
	}
	url := a.serverUrl + "/ota/devices/?"
	url += fmt.Sprintf("shared=%d&match_tag=%s&name_ilike=%s&factory=%s", sharedInt, matchTag, nameIlike, byFactory)
	logrus.Debugf("DeviceList with url: %s", url)
	return a.DeviceListCont(url)
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

func (a *Api) DeviceDelete(device string) error {
	bytes := []byte{}
	_, err := a.Delete(a.serverUrl+"/ota/devices/"+device+"/", bytes)
	return err
}

func (a *Api) DeviceListUpdates(device string) (*UpdateList, error) {
	return a.DeviceListUpdatesCont(a.serverUrl + "/ota/devices/" + device + "/updates/")
}

func (a *Api) DeviceListUpdatesCont(url string) (*UpdateList, error) {
	body, err := a.Get(url)
	if err != nil {
		return nil, err
	}

	updates := UpdateList{}
	err = json.Unmarshal(*body, &updates)
	if err != nil {
		return nil, err
	}
	return &updates, nil
}

func (a *Api) DeviceUpdateEvents(device, correlationId string) ([]UpdateEvent, error) {
	var events []UpdateEvent
	body, err := a.Get(a.serverUrl + "/ota/devices/" + device + "/updates/" + correlationId + "/")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(*body, &events)
	if err != nil {
		return events, err
	}
	return events, nil
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

func (a *Api) TargetUpdateTags(factory string, target_names []string, tag_names []string) (string, error) {
	type EmptyTarget struct {
		Custom TufCustom `json:"custom"`
	}
	tags := EmptyTarget{TufCustom{Tags: tag_names}}

	type Update struct {
		Targets map[string]EmptyTarget `json:"targets"`
	}
	update := Update{map[string]EmptyTarget{}}
	for idx := range target_names {
		update.Targets[target_names[idx]] = tags
	}

	data, err := json.Marshal(update)
	if err != nil {
		return "", err
	}

	url := a.serverUrl + "/ota/factories/" + factory + "/targets/"
	resp, err := a.Patch(url, data)
	if err != nil {
		return "", err
	}

	type PatchResp struct {
		JobServUrl string `json:"jobserv-url"`
	}
	pr := PatchResp{}
	if err := json.Unmarshal(*resp, &pr); err != nil {
		return "", err
	}
	return pr.JobServUrl + "runs/UpdateTargets/console.log", nil
}

func (a *Api) TargetDeleteTargets(factory string, target_names []string) (string, error) {
	type Update struct {
		Targets []string `json:"targets"`
	}
	update := Update{}
	update.Targets = target_names
	data, err := json.Marshal(update)
	if err != nil {
		return "", err
	}

	url := a.serverUrl + "/ota/factories/" + factory + "/targets/"
	resp, err := a.Delete(url, data)
	if err != nil {
		return "", err
	}

	type PatchResp struct {
		JobServUrl string `json:"jobserv-url"`
	}
	pr := PatchResp{}
	if err := json.Unmarshal(*resp, &pr); err != nil {
		return "", err
	}
	return pr.JobServUrl + "runs/UpdateTargets/console.log", nil
}

func (a *Api) JobservTail(url string) {
	offset := 0
	status := ""
	for {
		headers := map[string]string{"X-OFFSET": strconv.Itoa(offset)}
		resp, err := a.RawGet(url, &headers)
		if err != nil {
			fmt.Printf("TODO LOG ERROR OR SOMETHING: %s\n", err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Unable to read body resp: %s", err)
		}
		if resp.StatusCode != 200 {
			fmt.Printf("Unable to get '%s': HTTP_%d\n=%s", url, resp.StatusCode, body)
		}

		newstatus := resp.Header.Get("X-RUN-STATUS")
		if newstatus == "QUEUED" {
			if status == "" {
				os.Stdout.Write(body)
			} else {
				os.Stdout.WriteString(".")
			}
		} else if len(newstatus) == 0 {
			body = body[offset:]
			os.Stdout.Write(body)
			return
		} else {
			if newstatus != status {
				fmt.Printf("\n--- Status change: %s -> %s\n", status, newstatus)
			}
			os.Stdout.Write(body)
			offset += len(body)
		}
		status = newstatus
		time.Sleep(5 * time.Second)
	}
}
