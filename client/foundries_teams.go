package client

import "encoding/json"

type FactoryTeam struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type FactoryTeamDetails struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Scopes      []string      `json:"scopes"`
	Groups      []string      `json:"groups"`
	Members     []FactoryUser `json:"members"`
}

func (a *Api) TeamsList(factory string) ([]FactoryTeam, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/teams/"
	body, err := a.Get(url)
	if err != nil {
		return nil, err
	}

	var teams []FactoryTeam
	err = json.Unmarshal(*body, &teams)
	if err != nil {
		return nil, err
	}
	return teams, nil
}

func (a *Api) TeamDetails(factory string, team_name string) (*FactoryTeamDetails, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/teams/" + team_name
	body, err := a.Get(url)
	if err != nil {
		return nil, err
	}

	var team FactoryTeamDetails
	err = json.Unmarshal(*body, &team)
	if err != nil {
		return nil, err
	}
	return &team, nil
}
