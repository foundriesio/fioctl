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

func (a *Api) El2gDeleteDg(factory string) error {
	url := a.serverUrl + "/ota/factories/" + factory + "/el2g/"
	_, err := a.Delete(url, []byte(""))
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

type El2gDevice struct {
	DeviceGroup    string      `json:"device-group"`
	Id             json.Number `json:"id"`
	LastConnection string      `json:"last-connection"`
}

// TODO paginate
func (a *Api) El2gDevices(factory string) ([]El2gDevice, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/el2g/devices/"
	body, err := a.Get(url)
	if err != nil {
		return nil, err
	}
	var devices []El2gDevice
	if err = json.Unmarshal(*body, &devices); err != nil {
		return nil, err
	}
	return devices, nil
}

func (a *Api) El2gAddDevice(factory, prodId, deviceUuid string) error {
	url := a.serverUrl + "/ota/factories/" + factory + "/el2g/devices/"
	devices := []string{deviceUuid}

	type Req struct {
		ProductId string   `json:"product-id"`
		Devices   []string `json:"devices"`
	}

	body, err := json.Marshal(Req{prodId, devices})
	if err != nil {
		return err
	}
	_, err = a.Post(url, body)
	return err
}

type El2gProduct struct {
	Type string `json:"commercialName"`
	Nc12 string `json:"nc12"`
}

func (a *Api) El2gProductInfo(factory, deviceId string) (El2gProduct, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/el2g-proxy/devices/" + deviceId + "/product"
	body, err := a.Get(url)

	var prod El2gProduct
	if err != nil {
		return prod, err
	}
	if err = json.Unmarshal(*body, &prod); err != nil {
		return prod, err
	}
	return prod, nil
}

type El2gSecureObjectProvisioning struct {
	Name  string `json:"secureObjectName"`
	Type  string `json:"secureObjectType"`
	State string `json:"provisioningState"`
}

func (a *Api) El2gSecureObjectProvisionings(factory, deviceId string) ([]El2gSecureObjectProvisioning, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/el2g-proxy/rtp/devices/" + deviceId + "/secure-object-provisionings"
	body, err := a.Get(url)
	if err != nil {
		return nil, err
	}
	type resp struct {
		Content []El2gSecureObjectProvisioning `json:"content"`
	}
	var devices resp
	if err = json.Unmarshal(*body, &devices); err != nil {
		return nil, err
	}
	return devices.Content, nil
}
