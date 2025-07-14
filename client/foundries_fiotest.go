package client

import (
	"encoding/json"
	"strconv"

	"github.com/sirupsen/logrus"
)

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

type TargetTestingApi struct {
	api     Api
	factory string
}

func (a Api) TargetTestingApi(factory string) TargetTestingApi {
	return TargetTestingApi{api: a, factory: factory}
}

// Return a list of Targets that have been tested
func (a TargetTestingApi) Versions() ([]int, error) {
	url := a.api.serverUrl + "/ota/factories/" + a.factory + "/testing/"
	logrus.Debugf("TargetTesting with url: %s", url)

	body, err := a.api.Get(url)
	if err != nil {
		return nil, err
	}

	type resp struct {
		Versions []int `json:"versions"`
	}
	r := resp{}
	if err = json.Unmarshal(*body, &r); err != nil {
		return nil, err
	}
	return r.Versions, nil
}

func (a TargetTestingApi) Tests(target int) (*TargetTestList, error) {
	url := a.api.serverUrl + "/ota/factories/" + a.factory + "/targets/" + strconv.Itoa(target) + "/testing/"
	logrus.Debugf("TargetTests with url: %s", url)
	return a.api.TargetTestsCont(url)
}

func (a Api) TargetTestsCont(url string) (*TargetTestList, error) {
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

func (a TargetTestingApi) TestResults(target int, testId string) (*TargetTest, error) {
	url := a.api.serverUrl + "/ota/factories/" + a.factory + "/targets/" + strconv.Itoa(target) + "/testing/" + testId + "/"
	logrus.Debugf("TargetTests with url: %s", url)
	body, err := a.api.Get(url)
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

func (a TargetTestingApi) TestArtifact(target int, testId string, artifact string) (*[]byte, error) {
	url := a.api.serverUrl + "/ota/factories/" + a.factory + "/targets/" + strconv.Itoa(target) + "/testing/" + testId + "/" + artifact
	logrus.Debugf("TargetTests with url: %s", url)
	return a.api.Get(url)
}
