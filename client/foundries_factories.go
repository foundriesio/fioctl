package client

import (
	"encoding/json"
)

type Factory struct {
	Name string `json:"name"`
	Id   string `json:"reposerver-id"`
}

func (a *Api) FactoriesList(admin bool) ([]Factory, error) {
	url := a.serverUrl + "/ota/factories/"
	if admin {
		url += "?admin=1"
	}
	body, err := a.Get(url)
	if err != nil {
		return nil, err
	}
	var factories []Factory
	err = json.Unmarshal(*body, &factories)
	return factories, err
}
