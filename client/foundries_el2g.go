package client

import (
	"encoding/json"
)

type El2gCsr struct {
	Id    int    `json:"id"`
	Value string `json:"value"`
}

func (a *Api) El2gCreateDg(factory string) (El2gCsr, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/el2g/device-gateway/"
	body, err := a.Post(url, []byte(""))
	if err != nil {
		return El2gCsr{}, err
	}
	var csr El2gCsr
	err = json.Unmarshal(*body, &csr)
	return csr, err
}

func (a *Api) El2gUploadDgCert(factory string, caId int, rootCa, cert string) error {
	url := a.serverUrl + "/ota/factories/" + factory + "/el2g/device-gateway/"
	type Cert struct {
		Id                int    `json:"id"`
		RootCa            string `json:"root-ca"`
		SignedCertificate string `json:"signed-cert"`
	}

	body, err := json.Marshal(Cert{
		Id:                caId,
		RootCa:            rootCa,
		SignedCertificate: cert,
	})
	if err != nil {
		return err
	}
	_, err = a.Put(url, body)
	return err
}

type El2gAWSCert struct {
	CA   string `json:"ca"`
	Cert string `json:"cert"`
}

func (a *Api) El2gConfigAws(factory string, awsRegistrationCode string) (El2gAWSCert, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/el2g/aws-iot/"
	type Req struct {
		RegistrationCode string `json:"registration-code"`
	}

	body, err := json.Marshal(Req{awsRegistrationCode})
	if err != nil {
		return El2gAWSCert{}, err
	}
	resp, err := a.Post(url, body)
	if err != nil {
		return El2gAWSCert{}, err
	}
	var cert El2gAWSCert
	err = json.Unmarshal(*resp, &cert)
	return cert, err
}
