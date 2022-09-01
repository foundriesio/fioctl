package client

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
)

type El2gOverview struct {
	Subdomain  string `json:"subdomain"`
	ProductIds []int  `json:"product-ids"`
}

func (a *Api) El2gOverview(factory string) (El2gOverview, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/el2g/overview/"
	body, err := a.Get(url)
	if err != nil {
		return El2gOverview{}, err
	}
	var overview El2gOverview
	err = json.Unmarshal(*body, &overview)
	return overview, err
}

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

type El2gDevice struct {
	DeviceGroup    string      `json:"device-group"`
	Id             json.Number `json:"id"`
	LastConnection string      `json:"last-connection"`
}

func (a *Api) El2gDevices(factory string) ([]El2gDevice, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/el2g/devices/"
	logrus.Debugf("Getting el2g devices with: %s", url)
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

func (a *Api) El2gAddDevice(factory, prodId, deviceUuid string, production bool) error {
	url := a.serverUrl + "/ota/factories/" + factory + "/el2g/devices/"
	devices := []string{deviceUuid}

	type Req struct {
		ProductId  string   `json:"product-id"`
		Devices    []string `json:"devices"`
		Production bool     `json:"production"`
	}

	body, err := json.Marshal(Req{prodId, devices, production})
	if err != nil {
		return err
	}
	_, err = a.Post(url, body)
	return err
}

func (a *Api) El2gDeleteDevice(factory, prodId, deviceUuid string, production bool) error {
	url := a.serverUrl + "/ota/factories/" + factory + "/el2g/devices/"
	devices := []string{deviceUuid}

	type Req struct {
		ProductId  string   `json:"product-id"`
		Devices    []string `json:"devices"`
		Production bool     `json:"production"`
	}

	body, err := json.Marshal(Req{prodId, devices, production})
	if err != nil {
		return err
	}
	_, err = a.Delete(url, body)
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

type El2gIntermediateCa struct {
	Id        json.Number `json:"id"`
	Name      string      `json:"name"`
	Algorithm string      `json:"algorithm"`
	Value     string      `json:"value"`
}

func (a *Api) El2gIntermediateCas(factory string) ([]El2gIntermediateCa, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/el2g/intermediate-cas/"
	body, err := a.Get(url)

	var objs []El2gIntermediateCa
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(*body, &objs); err != nil {
		return nil, err
	}
	return objs, nil
}

type El2gSecureObject struct {
	Id       json.Number `json:"id"`
	Type     string      `json:"type"`
	Name     string      `json:"name"`
	ObjectId string      `json:"object-id"`
}

func (a *Api) El2gSecureObjects(factory string) ([]El2gSecureObject, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/el2g/secure-objects/"
	body, err := a.Get(url)

	var objs []El2gSecureObject
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(*body, &objs); err != nil {
		return nil, err
	}
	return objs, nil
}

type El2gSecureObjectProvisioning struct {
	Name  string `json:"secureObjectName"`
	Type  string `json:"secureObjectType"`
	State string `json:"provisioningState"`
	Cert  string `json:"certificate"`
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

func (a *Api) El2gProducts(factory string) ([]El2gProduct, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/el2g-proxy/products"
	body, err := a.Get(url)
	if err != nil {
		return nil, err
	}
	var products []El2gProduct
	if err = json.Unmarshal(*body, &products); err != nil {
		return nil, err
	}
	return products, nil
}
