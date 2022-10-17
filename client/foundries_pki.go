package client

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
)

type CaCerts struct {
	RootCrt string `json:"root-crt"`
	CaCrt   string `json:"ca-crt"`
	CaCsr   string `json:"ca-csr"`
	TlsCrt  string `json:"tls-crt"`
	TlsCsr  string `json:"tls-csr"`

	ChangeMeta ChangeMeta `json:"change-meta"`

	CreateCaScript       *string `json:"create_ca"`
	CreateDeviceCaScript *string `json:"create_device_ca"`
	SignCaScript         *string `json:"sign_ca_csr"`
	SignTlsScript        *string `json:"sign_tls_csr"`
}

func (a *Api) FactoryGetCA(factory string) (CaCerts, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/certs/"
	logrus.Debugf("Getting certs %s", url)
	var resp CaCerts

	body, err := a.Get(url)
	if err != nil {
		return resp, err
	}

	err = json.Unmarshal(*body, &resp)
	return resp, err
}

func (a *Api) FactoryCreateCA(factory string) (CaCerts, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/certs/"
	logrus.Debugf("Creating new factory CA %s", url)
	var resp CaCerts

	body, err := a.Post(url, []byte("{}"))
	if err != nil {
		return resp, err
	}

	err = json.Unmarshal(*body, &resp)
	return resp, err
}

func (a *Api) FactoryPatchCA(factory string, certs CaCerts) error {
	url := a.serverUrl + "/ota/factories/" + factory + "/certs/"
	logrus.Debugf("Patching factory CA %s", url)

	data, err := json.Marshal(certs)
	if err != nil {
		return err
	}

	_, err = a.Patch(url, data)
	return err
}
