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

	CaRevokeCrl   string   `json:"ca-revoke-crl,omitempty"`
	CaDisabled    []string `json:"disabled-ca-serials,omitempty"` // readonly
	ActiveRoot    string   `json:"active-root-serial,omitempty"`
	RootRevokeCrl string   `json:"root-revoke-crl,omitempty"`

	RootRenewalCorrelationId string `json:"root-renewal-correlation-id,omitempty"` // readonly

	ChangeMeta ChangeMeta `json:"change-meta"`
}

type CaCsrs struct {
	CaCsr  string `json:"ca-csr,omitempty"`
	EstCsr string `json:"est-tls-csr,omitempty"`
	TlsCsr string `json:"tls-csr,omitempty"`
}

type CaCreateOptions struct {
	FirstTimeInit  bool `json:"first-time-init"`
	CreateOnlineCa bool `json:"ca-csr"`
	CreateEstCert  bool `json:"est-tls-csr"`
	CreateTlsCert  bool `json:"tls-csr"`
}

func (a *Api) FactoryGetCA(factory string) (CaCerts, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/certs/"
	logrus.Debugf("Getting certs %s", url)
	var resp CaCerts

	body, err := a.Get(url)
	if err != nil {
		if herr := AsHttpError(err); herr != nil && herr.Response.StatusCode == 206 {
			err = errors.New("Factory PKI is not configured. Please, see `fioctl keys ca create`.")
		}
		return resp, err
	}

	err = json.Unmarshal(*body, &resp)
	return resp, err
}

func (a *Api) FactoryCreateCA(factory string, opts CaCreateOptions) (CaCsrs, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/certs/"
	logrus.Debugf("Creating new factory CA %s", url)
	var resp CaCsrs

	data, err := json.Marshal(opts)
	if err != nil {
		return resp, err
	}

	body, err := a.Post(url, data)
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

func (a *Api) FactoryEstUrl(factory string, port int, resource string) (string, error) {
	cert, err := a.FactoryGetCA(factory)
	if err != nil {
		return "", err
	}
	return cert.GetEstUrl(port, resource, false)
}

func (certs *CaCerts) GetEstUrl(port int, resource string, fallbackToGateway bool) (string, error) {
	crtPem := certs.EstCrt
	if len(crtPem) == 0 {
		if fallbackToGateway {
			// If the factory PKI was configured, the TLS cert is always present.
			crtPem = certs.TlsCrt
		} else {
			return "", errors.New("EST server is not configured. Please see `fioctl keys est`")
		}
	}

	block, _ := pem.Decode([]byte(crtPem))
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
