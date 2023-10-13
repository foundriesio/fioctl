package client

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
)

type CaCerts struct {
	RootCrt string `json:"root-crt,omitempty"`
	CaCrt   string `json:"ca-crt,omitempty"`
	EstCrt  string `json:"est-tls-crt,omitempty"`
	TlsCrt  string `json:"tls-crt,omitempty"`

	ChangeMeta ChangeMeta `json:"change-meta"`
}

type CaCsrs struct {
	CaCsr  string `json:"ca-csr,omitempty"`
	TlsCsr string `json:"tls-csr,omitempty"`
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

func (a *Api) FactoryCreateCA(factory string) (CaCsrs, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/certs/"
	logrus.Debugf("Creating new factory CA %s", url)
	var resp CaCsrs

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

type estCsr struct {
	TlsCsr string `json:"tls-csr"`
}
type estCrt struct {
	TlsCrt string `json:"tls-crt"`
}

func (a *Api) FactoryCreateEstCsr(factory string) (string, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/certs/est/"
	logrus.Debugf("Creating EST CSR %s", url)
	body, err := a.Post(url, nil)
	if err != nil {
		return "", err
	}
	var csr estCsr
	if err = json.Unmarshal(*body, &csr); err != nil {
		return "", err
	}
	return csr.TlsCsr, nil
}

func (a *Api) FactorySetEstCrt(factory string, cert string) error {
	url := a.serverUrl + "/ota/factories/" + factory + "/certs/est/"
	logrus.Debugf("Putting EST certs %s", url)
	crt := estCrt{cert}
	data, err := json.Marshal(crt)
	if err != nil {
		return err
	}
	_, err = a.Put(url, data)
	return err
}

func (a *Api) FactoryEstUrl(factory string, port int, resource string) (string, error) {
	cert, err := a.FactoryGetCA(factory)
	if err != nil {
		return "", err
	}
	if len(cert.EstCrt) == 0 {
		return "", errors.New("EST server is not configured. Please see `fioctl keys est`")
	}

	block, _ := pem.Decode([]byte(cert.EstCrt))
	c, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("Failed to parse certificate: %w", err)
	}
	if len(c.DNSNames) != 1 {
		return "", fmt.Errorf("Certificate expected to have 1 DNS name, %d found", len(c.DNSNames))
	}
	url := fmt.Sprintf("https://%s:%d%s", c.DNSNames[0], port, resource)
	return url, nil
}
