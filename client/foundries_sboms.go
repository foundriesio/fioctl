package client

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
)

type Sbom struct {
	CiBuild  string `json:"ci-build"`
	CiRun    string `json:"ci-run"`
	Artifact string `json:"artifact"`
	Uri      string `json:"uri"`
}

type SpdxPackage struct {
	Name             string
	LicenseConcluded string `json:"licenseConcluded"`
	LicenseDeclared  string `json:"licenseDeclared"`
	VersionInfo      string `json:"versionInfo"`
}

type SpdxDocument struct {
	Packages []SpdxPackage `json:"packages"`
}

func (a *Api) TargetSboms(factory string, targetName string) ([]Sbom, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/targets/" + targetName + "/sboms/"
	logrus.Debugf("TargetSboms with url: %s", url)

	body, err := a.Get(url)
	if err != nil {
		return nil, err
	}

	var sboms []Sbom
	if err = json.Unmarshal(*body, &sboms); err != nil {
		return nil, err
	}
	return sboms, nil
}

func (p SpdxPackage) License() string {
	if p.LicenseConcluded == p.LicenseDeclared {
		return p.LicenseConcluded
	} else if p.LicenseConcluded == "NOASSERTION" {
		return p.LicenseDeclared
	}
	return fmt.Sprintf("ERROR: Unknown license configuration for package: %s - Concluded(%s) Declared(%s)", p.Name, p.LicenseConcluded, p.LicenseDeclared)
}

func (a *Api) SbomDownload(factory, targetName, path, contentType string) ([]byte, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/targets/" + targetName + "/sboms/" + path
	logrus.Debugf("SbomDownload with %s url: %s", contentType, url)
	headers := map[string]string{"accept": contentType}
	res, err := a.RawGet(url, &headers)
	if err != nil {
		return nil, err
	}
	buf, err := readResponse(res)
	if err != nil {
		return nil, err
	}
	return *buf, nil
}
