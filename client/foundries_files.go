package client

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"

	"github.com/sirupsen/logrus"
)

func (a *Api) CreateFile(factory string, content []byte) (string, error) {
	url := a.serverUrl + "/ota/factories/" + factory + "/files/"
	logrus.Debugf("Uploading file to: %s", url)

	sum := sha256.Sum256(content)
	sumStr := fmt.Sprintf("%x", sum)

	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)
	if err := w.WriteField("sha256", sumStr); err != nil {
		return "", err
	}
	f, err := w.CreateFormFile("file", "file")
	if err != nil {
		return "", err
	}
	if _, err = f.Write(content); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(buf.Bytes()))
	if err != nil {
		return "", err
	}

	a.setReqHeaders(req, true)
	req.Header.Set("Content-type", w.FormDataContentType())

	log := httpLogger(req)
	res, err := a.client.Do(req)
	if err != nil {
		log.Debugf("Network Error: %s", err)
		return "", err
	}
	_, err = readResponse(res, log)
	return sumStr, err
}

type MCUTarget struct {
	Sha256  string   `json:"sha256"`
	Version int      `json:"version"`
	HwId    string   `json:"hwid"`
	Tags    []string `json:"tags"`
}

func (a *Api) CreateMcuTarget(factory string, target MCUTarget) error {
	url := a.serverUrl + "/ota/factories/" + factory + "/targets/mcu-firmware/"
	logrus.Debugf("Creating target %v", target)
	bytes, err := json.Marshal(target)
	if err != nil {
		return err
	}
	_, err = a.Post(url, bytes)
	return err
}
