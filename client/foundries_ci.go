package client

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
)

type Worker struct {
	Name           string   `json:"name"`
	Enlisted       bool     `json:"enlisted"`
	Online         bool     `json:"online"`
	Surges         bool     `json:"surges"`
	HostTags       []string `json:"host_tags"`
	ConcurrentRuns int      `json:"concurrent_runs"`
	CpuType        string   `json:"cpu_type"`
	CpuTotal       int      `json:"cpu_total"`
	MemTotal       int      `json:"mem_total"`
	Distro         string   `json:"distro"`
}

func (a *Api) CiWorkersList() ([]Worker, error) {
	url := a.serverUrl + "/workers/"
	type response struct {
		Data struct {
			Next    string   `json:"next"`
			Workers []Worker `json:"workers"`
		} `json:"data"`
	}
	var workers []Worker
	for len(url) > 0 {
		logrus.Debugf("Getting workers %s", url)
		body, err := a.Get(url)
		if err != nil {
			return nil, err
		}
		var resp response
		if err = json.Unmarshal(*body, &resp); err != nil {
			return nil, err
		}
		if len(resp.Data.Workers) > 0 {
			workers = append(workers, resp.Data.Workers...)
		}
		url = resp.Data.Next
	}
	return workers, nil
}
