package client

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
)

type Sbom struct {
	CiBuild  string `json:"ci-build"`
	CiRun    string `json:"ci-run"`
	Artifact string `json:"artifact"`
	Uri      string `json:"uri"`
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
