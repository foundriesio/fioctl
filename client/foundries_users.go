package client

import "encoding/json"

type FactoryUser struct {
	PolisId string `json:"polis-id"`
	Name    string `json:"name"`
	Role    string `json:"role"`
}

type FactoryUserAccessDetails struct {
	PolisId         string               `json:"polis-id"`
	Name            string               `json:"name"`
	Role            string               `json:"role"`
	Teams           []FactoryTeamDetails `json:"teams-ext"`
	EffectiveScopes []string             `json:"effective-scopes"`
}

func (a *Api) UsersList(factory string) ([]FactoryUser, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/users/"
	body, err := a.Get(url)
	if err != nil {
		return nil, err
	}

	var users []FactoryUser
	err = json.Unmarshal(*body, &users)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (a *Api) UserAccessDetails(factory string, user_id string) (*FactoryUserAccessDetails, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/users/" + user_id + "/"
	body, err := a.Get(url)
	if err != nil {
		return nil, err
	}

	var user FactoryUserAccessDetails
	err = json.Unmarshal(*body, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
