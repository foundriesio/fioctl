package client

import (
	"bytes"
	"encoding/base64"
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

type FactoryUser struct {
	PolisId string `json:"polis-id"`
	Name    string `json:"name"`
	Role    string `json:"role"`
}

type ProjectSecret struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ProjectTrigger struct {
	Type    string          `json:"type"`
	Id      int             `json:"id,omitempty"`
	Secrets []ProjectSecret `json:"secrets"`
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

func (a *Api) setReqHeaders(req *http.Request, jsonContent bool) {
	req.Header.Set("User-Agent", "fioctl")

	if len(a.config.Token) > 0 {
		logrus.Debug("Using API token for http request")
		req.Header.Set("OSF-TOKEN", a.config.Token)
	}

	if len(a.config.ClientCredentials.AccessToken) > 0 {
		logrus.Debug("Using oauth token for http request")
		tok := base64.StdEncoding.EncodeToString([]byte(a.config.ClientCredentials.AccessToken))
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	if jsonContent {
		req.Header.Set("Content-Type", "application/json")
	}
}

func (a *Api) RawGet(url string, headers *map[string]string) (*http.Response, error) {
	client := http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	a.setReqHeaders(req, false)
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

	a.setReqHeaders(req, true)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 202 || res.StatusCode != 200 {
		return nil, fmt.Errorf("Unable to PATCH '%s': HTTP_%d\n=%s", url, res.StatusCode, body)
	}
	return &body, nil
}

func (a *Api) Post(url string, data []byte) (*[]byte, error) {
	client := http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	a.setReqHeaders(req, true)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 201 {
		return nil, fmt.Errorf("Unable to POST '%s': HTTP_%d\n=%s", url, res.StatusCode, body)
	}
	return &body, nil
}

func (a *Api) Put(url string, data []byte) (*[]byte, error) {
	client := http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	a.setReqHeaders(req, true)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 202 {
		return nil, fmt.Errorf("Unable to PUT '%s': HTTP_%d\n=%s", url, res.StatusCode, body)
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

	a.setReqHeaders(req, true)

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

func (a *Api) TargetsPut(factory string, data []byte) (string, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/targets/"
	resp, err := a.Put(url, data)
	if err != nil {
		return "", err
	}
	type PutResp struct {
		JobServUrl string `json:"jobserv-url"`
	}
	pr := PutResp{}
	if err := json.Unmarshal(*resp, &pr); err != nil {
		return "", err
	}
	return pr.JobServUrl + "runs/UpdateTargets/console.log", nil
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

func (a *Api) FactoryTriggers(factory string) ([]ProjectTrigger, error) {
	type Resp struct {
		Data []ProjectTrigger `json:"data"`
	}

	body, err := a.Get(a.serverUrl + "/projects/" + factory + "/lmp/triggers/")
	if err != nil {
		return nil, err
	}
	r := Resp{}
	err = json.Unmarshal(*body, &r)
	return r.Data, err
}

func (a *Api) FactoryUpdateTrigger(factory string, t ProjectTrigger) error {
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}

	url := a.serverUrl + "/projects/" + factory + "/lmp/triggers/"
	if t.Id == 0 {
		logrus.Debugf("Creating new trigger")
		_, err := a.Post(url, data)
		return err
	} else {
		logrus.Debugf("Patching trigger %d", t.Id)
		url += strconv.Itoa(t.Id) + "/"
		_, err := a.Patch(url, data)
		return err
	}
}

func (a *Api) UsersList(factory string) ([]FactoryUser, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/users/"
	body, err := a.Get(url)
	if err != nil {
		return nil, err
	}

	var users []FactoryUser
	err = json.Unmarshal(*body, &users)
	if err != nil {
		return nil, err
	}
	return users, nil
}
