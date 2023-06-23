package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const OauthURL string = "https://app.foundries.io/oauth"

type OAuthConfig struct {
	ClientId     string  `mapstructure:"client_id"`
	ClientSecret string  `mapstructure:"client_secret"`
	TokenType    string  `mapstructure:"token_type"`
	AccessToken  string  `mapstructure:"access_token"`
	RefreshToken string  `mapstructure:"refresh_token"`
	ExpiresIn    float64 `mapstructure:"expires_in"`
	Created      string
	DefaultOrg   string
	URL          string
}

type ClientCredentials struct {
	Config OAuthConfig
}

type Org struct {
	Name string `json:"name"`
}

type OAuthResponse struct {
	TokenType    string  `json:"token_type"`
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	ExpiresIn    float64 `json:"expires_in"`
	Scope        string  `json:"scope"`
	Orgs         []Org   `json:"orgs"`
}

func buildUrl(uri string, p string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}

	u.Path = path.Join(u.Path, p)
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}

	return u.String(), nil
}

func (c *ClientCredentials) updateConfig(r OAuthResponse) {
	c.Config.AccessToken = r.AccessToken
	c.Config.RefreshToken = r.RefreshToken
	c.Config.TokenType = r.TokenType
	c.Config.ExpiresIn = r.ExpiresIn
	c.Config.Created = time.Now().UTC().Format(time.RFC3339)
	if len(r.Orgs) == 1 {
		c.Config.DefaultOrg = r.Orgs[0].Name
	}
}

// Perform a POST request.
func (c *ClientCredentials) post(uri string, data url.Values) (*[]byte, error) {
	res, err := http.PostForm(uri, data)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Unable to retrieve credentials: HTTP_%d\n=%s", res.StatusCode, body)
	}

	return &body, nil
}

// IsExpired checks if the OAuth token is expired.
func (c *ClientCredentials) IsExpired() (bool, error) {
	if c.Config.ExpiresIn != 0 && c.Config.Created != "" {
		t, err := time.Parse(time.RFC3339, c.Config.Created)
		if err != nil {
			return false, fmt.Errorf("Error parsing oauth token creation date")
		}

		if time.Since(t).Seconds() > c.Config.ExpiresIn {
			return true, nil
		}
	}

	return false, nil
}

// HasRefreshToken checks if a refresh token is available.
func (c *ClientCredentials) HasRefreshToken() bool {
	return c.Config.RefreshToken != ""
}

// Refresh performs a refresh of the token.
func (c *ClientCredentials) Refresh() error {
	logrus.Debug("Refreshing client_credentials oauth token")

	u, err := buildUrl(c.Config.URL, "token")
	if err != nil {
		return err
	}

	logrus.Debug("Refreshing oauth token via URL ", u)

	v := url.Values{}
	v.Set("grant_type", "refresh_token")
	v.Set("client_id", c.Config.ClientId)
	v.Set("client_secret", c.Config.ClientSecret)
	v.Set("refresh_token", c.Config.RefreshToken)

	data, err := c.post(u, v)
	if err != nil {
		return err
	}

	t := OAuthResponse{}
	err = json.Unmarshal(*data, &t)
	if err != nil {
		return err
	}

	c.updateConfig(t)
	return nil
}

// Get a new token.
func (c *ClientCredentials) Get() error {
	logrus.Debug("Getting oauth token via client_credentials grant")

	u, err := buildUrl(c.Config.URL, "token")
	if err != nil {
		return err
	}

	logrus.Debug("Getting oauth token via URL ", u)

	v := url.Values{}
	v.Set("grant_type", "client_credentials")
	v.Set("client_id", c.Config.ClientId)
	v.Set("client_secret", c.Config.ClientSecret)

	data, err := c.post(u, v)
	if err != nil {
		return err
	}

	t := OAuthResponse{}
	err = json.Unmarshal(*data, &t)
	if err != nil {
		return err
	}

	c.updateConfig(t)
	return nil
}

func NewClientCredentials(c OAuthConfig) ClientCredentials {
	if len(c.URL) == 0 {
		c.URL = OauthURL
	}
	return ClientCredentials{c}
}
