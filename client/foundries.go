package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
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
	client    http.Client
}

type ConfigFile struct {
	Name        string   `json:"name"`
	Value       string   `json:"value"`
	Unencrypted bool     `json:"unencrypted"`
	OnChanged   []string `json:"on-changed,omitempty"`
}

type ConfigCreateRequest struct {
	Reason string       `json:"reason"`
	Files  []ConfigFile `json:"files"`
}

type DeviceConfig struct {
	CreatedAt string       `json:"created-at"`
	AppliedAt string       `json:"applied-at"`
	Reason    string       `json:"reason"`
	Files     []ConfigFile `json:"files"`
}

type DeviceConfigList struct {
	Configs []DeviceConfig `json:"config"`
	Total   int            `json:"total"`
	Next    *string        `json:"next"`
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
	UpToDate      bool             `json:"up-to-date"`
	PublicKey     string           `json:"public-key"`
	ActiveConfig  *DeviceConfig    `json:"active-config,omitempty"`
	AktualizrToml string           `json:"aktualizr-toml,omitempty"`
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

type ComposeApp struct {
	Uri string `json:"uri"`
}

type FactoryUser struct {
	PolisId string `json:"polis-id"`
	Name    string `json:"name"`
	Role    string `json:"role"`
}

type TargetStatus struct {
	Devices int `json:"devices"`
	Version int `json:"version"`
}
type TagStatus struct {
	Name            string         `json:"name"`
	DevicesTotal    int            `json:"devices-total"`
	DevicesOnline   int            `json:"devices-online"`
	DevicesOnLatest int            `json:"devices-on-latest"`
	LatestTarget    int            `json:"latest-target"`
	Targets         []TargetStatus `json:"targets"`
}

type FactoryStatus struct {
	TotalDevices int         `json:"total-devices"`
	Tags         []TagStatus `json:"tags"`
}

type ProjectSecret struct {
	Name  string  `json:"name"`
	Value *string `json:"value"`
}

type ProjectTrigger struct {
	Type    string          `json:"type"`
	Id      int             `json:"id,omitempty"`
	Secrets []ProjectSecret `json:"secrets"`
}

type TufCustom struct {
	HardwareIds    []string              `json:"hardwareIds,omitempty"`
	Tags           []string              `json:"tags,omitempty"`
	TargetFormat   string                `json:"targetFormat,omitempty"`
	Version        string                `json:"version,omitempty"`
	DockerApps     map[string]DockerApp  `json:"docker_apps,omitempty"`
	ComposeApps    map[string]ComposeApp `json:"docker_compose_apps,omitempty"`
	Name           string                `json:"name,omitempty"`
	ContainersSha  string                `json:"containers-sha,omitempty"`
	LmpManifestSha string                `json:"lmp-manifest-sha,omitempty"`
	OverridesSha   string                `json:"meta-subscriber-overrides-sha,omitempty"`
}

type TargetTestResults struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Details string `json:"details"`
}

type TargetTest struct {
	Name        string              `json:"name"`
	Id          string              `json:"test-id"`
	DeviceUUID  string              `json:"device-uuid"`
	DeviceName  string              `json:"device-name"`
	Status      string              `json:"status"`
	Details     string              `json:"details"`
	CreatedOn   float32             `json:"created-on"`
	CompletedOn float32             `json:"completed-on"`
	Results     []TargetTestResults `json:"results"`
	Artifacts   []string            `json:"artifacts"`
}

type TargetTestList struct {
	Tests []TargetTest `json:"tests"`
	Total int          `json:"total"`
	Next  *string      `json:"next"`
}

func (d Device) Online(inactiveHoursThreshold int) bool {
	if len(d.LastSeen) == 0 {
		return false
	}
	t, err := time.Parse("2006-01-02T15:04:05+00:00", d.LastSeen)
	if err == nil {
		duration := time.Since(t)
		if duration.Hours() > float64(inactiveHoursThreshold) {
			return false
		}
	} else {
		logrus.Error(err)
		return false
	}
	return true
}

func NewApiClient(serverUrl string, config Config, caCertPath string) *Api {
	var tlsConfig *tls.Config
	if len(caCertPath) > 0 {
		rootCAs, _ := x509.SystemCertPool()
		if rootCAs == nil {
			rootCAs = x509.NewCertPool()
		}

		certs, err := ioutil.ReadFile(caCertPath)
		if err != nil {
			logrus.Fatalf("Failed to append %q to RootCAs: %v", caCertPath, err)
		}

		if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
			logrus.Warning("No certs appended, using system certs only")
		}

		tlsConfig = &tls.Config{
			RootCAs: rootCAs,
		}
	}
	api := Api{
		serverUrl: strings.TrimRight(serverUrl, "/"),
		config:    config,
		client: http.Client{
			Timeout: time.Second * 10,
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		},
	}
	return &api
}

func getResponse(resp *[]byte, err error, runName string) (string, string, error) {
	if err != nil {
		return "", "", err
	}
	type PatchResp struct {
		JobServUrl string `json:"jobserv-url"`
		WebUrl     string `json:"web-url"`
	}
	pr := PatchResp{}
	if err := json.Unmarshal(*resp, &pr); err != nil {
		return "", "", err
	}
	return pr.JobServUrl + fmt.Sprintf("runs/%s/console.log", runName), pr.WebUrl, nil
}

func (a *Api) setReqHeaders(req *http.Request, jsonContent bool) {
	req.Header.Set("User-Agent", "fioctl-2")

	if len(a.config.Token) > 0 {
		logrus.Debug("Using API token for http request")
		headerName := os.Getenv("TOKEN_HEADER")
		if len(headerName) == 0 {
			headerName = "OSF-TOKEN"
		}
		req.Header.Set(headerName, a.config.Token)
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

	return a.client.Do(req)
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
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	a.setReqHeaders(req, true)

	res, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 202 && res.StatusCode != 201 && res.StatusCode != 200 {
		return nil, fmt.Errorf("Unable to PATCH '%s': HTTP_%d\n=%s", url, res.StatusCode, body)
	}
	return &body, nil
}

func (a *Api) Post(url string, data []byte) (*[]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	a.setReqHeaders(req, true)

	res, err := a.client.Do(req)
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
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	a.setReqHeaders(req, true)

	res, err := a.client.Do(req)
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
	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	a.setReqHeaders(req, true)

	res, err := a.client.Do(req)
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

func (a *Api) DeviceCreateConfig(device string, cfg ConfigCreateRequest) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	url := a.serverUrl + "/ota/devices/" + device + "/config/"
	logrus.Debug("Creating new device config")
	_, err = a.Post(url, data)
	return err
}

func (a *Api) DevicePatchConfig(device string, cfg ConfigCreateRequest) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	url := a.serverUrl + "/ota/devices/" + device + "/config/"
	logrus.Debug("Patching device config")
	_, err = a.Patch(url, data)
	return err
}

func (a *Api) DeviceListConfig(device string) (*DeviceConfigList, error) {
	url := a.serverUrl + "/ota/devices/" + device + "/config/"
	logrus.Debugf("DeviceListConfig with url: %s", url)
	return a.DeviceListConfigCont(url)
}

func (a *Api) DeviceListConfigCont(url string) (*DeviceConfigList, error) {
	body, err := a.Get(url)
	if err != nil {
		return nil, err
	}

	config := DeviceConfigList{}
	err = json.Unmarshal(*body, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (a *Api) DeviceDeleteConfig(device, filename string) error {
	url := a.serverUrl + "/ota/devices/" + device + "/config/" + filename + "/"
	logrus.Debugf("Deleting config file: %s", url)
	_, err := a.Delete(url, nil)
	return err
}

func (a *Api) FactoryCreateConfig(factory string, cfg ConfigCreateRequest) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	url := a.serverUrl + "/ota/factories/" + factory + "/config/"
	logrus.Debug("Creating new factory config")
	_, err = a.Post(url, data)
	return err
}

func (a *Api) FactoryDeleteConfig(factory, filename string) error {
	url := a.serverUrl + "/ota/factories/" + factory + "/config/" + filename + "/"
	logrus.Debugf("Deleting config file: %s", url)
	_, err := a.Delete(url, nil)
	return err
}

func (a *Api) FactoryPatchConfig(factory string, cfg ConfigCreateRequest) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	url := a.serverUrl + "/ota/factories/" + factory + "/config/"
	logrus.Debug("Creating new factory config")
	_, err = a.Patch(url, data)
	return err
}

func (a *Api) FactoryListConfig(factory string) (*DeviceConfigList, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/config/"
	logrus.Debugf("FactoryListConfig with url: %s", url)
	return a.DeviceListConfigCont(url)
}

func (a *Api) FactoryListConfigCont(url string) (*DeviceConfigList, error) {
	return a.DeviceListConfigCont(url)
}

func (a *Api) FactoryStatus(factory string, inactiveThreshold int) (*FactoryStatus, error) {
	url := fmt.Sprintf("%s/ota/factories/%s/status/?offline-threshold=%d", a.serverUrl, factory, inactiveThreshold)
	logrus.Debugf("FactoryStatus with url: %s", url)
	body, err := a.Get(url)
	if err != nil {
		return nil, err
	}
	s := FactoryStatus{}
	err = json.Unmarshal(*body, &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
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

func (a *Api) TargetsPut(factory string, data []byte) (string, string, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/targets/"
	resp, err := a.Put(url, data)
	if err != nil {
		return "", "", err
	}
	return getResponse(resp, err, "UpdateTargets")
}

func (a *Api) TargetUpdateTags(factory string, target_names []string, tag_names []string) (string, string, error) {
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
		return "", "", err
	}

	url := a.serverUrl + "/ota/factories/" + factory + "/targets/"
	resp, err := a.Patch(url, data)
	return getResponse(resp, err, "UpdateTargets")
}

func (a *Api) TargetDeleteTargets(factory string, target_names []string) (string, string, error) {
	type Update struct {
		Targets []string `json:"targets"`
	}
	update := Update{}
	update.Targets = target_names
	data, err := json.Marshal(update)
	if err != nil {
		return "", "", err
	}

	url := a.serverUrl + "/ota/factories/" + factory + "/targets/"
	resp, err := a.Delete(url, data)
	return getResponse(resp, err, "UpdateTargets")
}

func (a *Api) TargetImageCreate(factory string, targetName string, appShortlist string) (string, string, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/targets/" + targetName + "/images/"
	if len(appShortlist) > 0 {
		url += "?app_shortlist=" + appShortlist
	}
	resp, err := a.Post(url, nil)
	return getResponse(resp, err, "assemble-system-image")
}

func (a *Api) TargetTests(factory string, target int) (*TargetTestList, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/targets/" + strconv.Itoa(target) + "/testing/"
	logrus.Debugf("TargetTests with url: %s", url)
	return a.TargetTestsCont(url)
}

func (a *Api) TargetTestsCont(url string) (*TargetTestList, error) {
	body, err := a.Get(url)
	if err != nil {
		return nil, err
	}

	tests := TargetTestList{}
	err = json.Unmarshal(*body, &tests)
	if err != nil {
		return nil, err
	}
	return &tests, nil
}

func (a *Api) TargetTestResults(factory string, target int, testId string) (*TargetTest, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/targets/" + strconv.Itoa(target) + "/testing/" + testId + "/"
	logrus.Debugf("TargetTests with url: %s", url)
	body, err := a.Get(url)
	if err != nil {
		return nil, err
	}

	test := TargetTest{}
	err = json.Unmarshal(*body, &test)
	if err != nil {
		return nil, err
	}
	return &test, nil
}

func (a *Api) TargetTestArtifact(factory string, target int, testId string, artifact string) (*[]byte, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/targets/" + strconv.Itoa(target) + "/testing/" + testId + "/" + artifact
	logrus.Debugf("TargetTests with url: %s", url)
	return a.Get(url)
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
