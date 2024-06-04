package client

import (
	"encoding/json"
	"errors"
	"fmt"
	netUrl "net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type DeviceConfig struct {
	CreatedAt string       `json:"created-at"`
	AppliedAt string       `json:"applied-at"` // This is not present in factory config
	CreatedBy string       `json:"created-by"`
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
	Details    string `json:"details"`
}

type UpdateEvent struct {
	Time   string      `json:"deviceTime"`
	Type   EventType   `json:"eventType"`
	Detail EventDetail `json:"event"`
}

type AppServiceState struct {
	Name     string `json:"name"`
	Hash     string `json:"hash"`
	State    string `json:"state"`
	Status   string `json:"status"`
	Health   string `json:"health"`
	ImageUri string `json:"image"`
	Logs     string `json:"logs"`
}

type AppState struct {
	Services []AppServiceState `json:"services"`
	State    string            `json:"state"`
	Uri      string            `json:"uri"`
}

type AppsState struct {
	Ostree     string              `json:"ostree"`
	DeviceTime string              `json:"deviceTime"`
	Apps       map[string]AppState `json:"apps,omitempty"`
}

type AppsStates struct {
	States []AppsState `json:"apps-states"`
}

type Device struct {
	Uuid          string           `json:"uuid"`
	Name          string           `json:"name"`
	Owner         string           `json:"owner"`
	Factory       string           `json:"factory"`
	GroupName     string           `json:"device-group"` // Returned in List API
	Group         *DeviceGroup     `json:"group"`        // Returned in Get API
	LastSeen      string           `json:"last-seen"`
	OstreeHash    string           `json:"ostree-hash"`
	LmpVer        string           `json:"lmp-ver,omitempty"`
	DockerApps    []string         `json:"docker-apps,omitempty"`
	Tag           string           `json:"tag,omitempty"`
	Network       *NetInfo         `json:"network-info,omitempty"`
	Hardware      *json.RawMessage `json:"hardware-info,omitempty"`
	TargetName    string           `json:"target-name"`
	Status        string           `json:"status"`
	CurrentUpdate string           `json:"current-update"`
	UpToDate      bool             `json:"up-to-date"`
	PublicKey     string           `json:"public-key"`
	ActiveConfig  *DeviceConfig    `json:"active-config,omitempty"`
	AktualizrToml string           `json:"aktualizr-toml,omitempty"`
	IsProd        bool             `json:"is-prod"`
	IsWave        bool             `json:"is-wave"`
	ChangeMeta    ChangeMeta       `json:"change-meta"`
	Secondaries   []struct {
		Serial     string `json:"serial"`
		TargetName string `json:"target-name"`
		HardwareId string `json:"hardware-id"`
	} `json:"secondary-ecus"`
	AppsState *AppsState `json:"apps-state,omitempty"`
}

type DeviceApi struct {
	api     *Api
	factory string
	id      string
	byUuid  bool
}

func (a *Api) DeviceApiByName(factory, name string) DeviceApi {
	return DeviceApi{
		api:     a,
		factory: factory,
		id:      name,
		byUuid:  false,
	}
}

func (a *Api) DeviceApiByUuid(factory, uuid string) DeviceApi {
	return DeviceApi{
		api:     a,
		factory: factory,
		id:      uuid,
		byUuid:  true,
	}
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

func (a ComposeApp) Hash() string {
	parts := strings.SplitN(a.Uri, "@sha256:", 2)
	return parts[len(parts)-1]
}

func (a ComposeApp) Name() string {
	parts := strings.SplitN(a.Uri, "@sha256:", 2)
	nameStartIndx := strings.LastIndexByte(parts[0], '/')
	return parts[0][nameStartIndx+1:]
}

func (d Device) Online(inactiveHoursThreshold int) bool {
	if len(d.LastSeen) == 0 {
		return false
	}
	t, err := time.Parse(time.RFC3339, d.LastSeen)
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

func (a *Api) DeviceGet(factory, device string) (*Device, error) {
	url := a.serverUrl + "/ota/devices/" + device + "/?factory=" + factory
	body, err := a.Get(url)
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

func (a *Api) DeviceList(filterBy map[string]string, sortBy string, page, limit uint64) (*DeviceList, error) {
	url := a.serverUrl + "/ota/devices/?"
	query := netUrl.Values{}
	for key, val := range filterBy {
		if len(val) > 0 {
			query.Set(key, val)
		}
	}
	if len(sortBy) > 0 {
		query.Set("sortby", sortBy)
	}
	if page > 1 {
		query.Set("page", strconv.FormatUint(page, 10))
	}
	query.Set("limit", strconv.FormatUint(limit, 10))
	url += query.Encode()
	logrus.Debugf("DeviceList with url: %s", url)
	return a.DeviceListCont(url)
}

func (a *Api) DeviceListCont(url string) (*DeviceList, error) {
	logrus.Debugf("DeviceListCont with url: %s", url)
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

func (a *Api) DeviceListDenied(factory string, page, limit uint64) (*DeviceList, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/denied-devices/"
	url += fmt.Sprintf("?limit=%d&page=%d", limit, page)
	return a.DeviceListCont(url)
}

func (d *DeviceApi) url(resource string) string {
	url := d.api.serverUrl + "/ota/devices/" + d.id
	url += resource
	url += "?factory=" + d.factory
	if d.byUuid {
		url += "&by-uuid=1"
	}
	return url
}

func (d *DeviceApi) Chown(owner string) error {
	body := map[string]string{"owner": owner}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	_, err = d.api.Patch(d.url("/"), data)
	return err
}

func (d *DeviceApi) Rename(newName string) error {
	body := map[string]string{"name": newName}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	_, err = d.api.Patch(d.url("/"), data)
	return err
}

func (d *DeviceApi) SetGroup(group string) error {
	body := map[string]string{"group": group}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	_, err = d.api.Patch(d.url("/"), data)
	return err
}

func (d *DeviceApi) Delete() error {
	bytes := []byte{}
	_, err := d.api.Delete(d.url("/"), bytes)
	return err
}

func (d *DeviceApi) DeleteDenied() error {
	if !d.byUuid {
		return errors.New("DeviceApi for DeleteDenied requires a UUID and not a name")
	}
	bytes := []byte{}
	url := d.api.serverUrl + "/ota/factories/" + d.factory + "/denied-devices/" + d.id + "/"
	_, err := d.api.Delete(url, bytes)
	return err
}

func (d *DeviceApi) ListUpdates() (*UpdateList, error) {
	return d.ListUpdatesCont(d.url("/updates/"))
}

func (d *DeviceApi) ListUpdatesCont(url string) (*UpdateList, error) {
	body, err := d.api.Get(url)
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

func (d *DeviceApi) UpdateEvents(correlationId string) ([]UpdateEvent, error) {
	var events []UpdateEvent
	body, err := d.api.Get(d.url("/updates/" + correlationId + "/"))
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(*body, &events)
	if err != nil {
		return events, err
	}
	return events, nil
}

func (a *Api) DeviceCreateConfig(factory, device string, cfg ConfigCreateRequest) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	url := a.serverUrl + "/ota/devices/" + device + "/config/?factory=" + factory
	logrus.Debug("Creating new device config")
	_, err = a.Post(url, data)
	return err
}

func (a *Api) DevicePatchConfig(factory, device string, cfg ConfigCreateRequest, force bool) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	url := a.serverUrl + "/ota/devices/" + device + "/config/?factory=" + factory
	if force {
		url += "&force=1"
	}
	logrus.Debug("Patching device config")
	_, err = a.Patch(url, data)
	return err
}

func (a *Api) DeviceListConfig(factory, device string) (*DeviceConfigList, error) {
	url := a.serverUrl + "/ota/devices/" + device + "/config/?factory=" + factory
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

func (a *Api) DeviceDeleteConfig(factory, device, filename string) error {
	url := a.serverUrl + "/ota/devices/" + device + "/config/" + filename + "/?factory=" + factory
	logrus.Debugf("Deleting config file: %s", url)
	_, err := a.Delete(url, nil)
	return err
}

func (d *DeviceApi) GetAppsStates() (*AppsStates, error) {
	body, err := d.api.Get(d.url("/apps-states/"))
	if err != nil {
		return nil, err
	}

	states := AppsStates{}
	err = json.Unmarshal(*body, &states)
	if err != nil {
		return nil, err
	}
	return &states, nil
}
