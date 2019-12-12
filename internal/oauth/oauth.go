package oauth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/foundriesio/fioctl/internal/config"
	"github.com/sirupsen/logrus"
)

const uri string = "https://app.foundries.io/oauth"

// OAuth interface.
type OAuth interface {
	Get() int
	HasRefreshToken() bool
	IsExpired() (bool, error)
	Refresh() int
	post(string, url.Values) (*[]byte, error)
}

// The ClientCredentials struct implements the OAuth interface for the
// client_credentials/refresh_token OAuth grant types.
type ClientCredentials struct {
	Config *config.Config
	URL    string
}

type OAuthResponse struct {
	TokenType    string  `json:"token_type"`
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	ExpiresIn    float64 `json:"expires_in"`
	Scope        string  `json:"scope"`
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

func updateConfig(c *config.Config, r *OAuthResponse) {
	c.ClientCredentials.AccessToken = r.AccessToken
	c.ClientCredentials.RefreshToken = r.RefreshToken
	c.ClientCredentials.TokenType = r.TokenType
	c.ClientCredentials.ExpiresIn = r.ExpiresIn
	c.ClientCredentials.Created = time.Now().UTC().Format(time.RFC3339)
}

// Perform a POST request.
func (c *ClientCredentials) post(uri string, data url.Values) (*[]byte, error) {
	res, err := http.PostForm(uri, data)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
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
	if c.Config.ClientCredentials.ExpiresIn != 0 && c.Config.ClientCredentials.Created != "" {
		t, err := time.Parse(time.RFC3339, c.Config.ClientCredentials.Created)
		if err != nil {
			return false, fmt.Errorf("Error parsing oauth token creation date")
		}

		if time.Since(t).Seconds() > c.Config.ClientCredentials.ExpiresIn {
			return true, nil
		}
	}

	return false, nil
}

// HasRefreshToken checks if a refresh token is available.
func (c *ClientCredentials) HasRefreshToken() bool {
	if c.Config.ClientCredentials.RefreshToken != "" {
		return true
	}
	return false
}

// Refresh performs a refresh of the token.
func (c *ClientCredentials) Refresh() int {
	logrus.Debug("Refreshing client_credentials oauth token")

	u, err := buildUrl(c.URL, "token")
	if err != nil {
		fmt.Println("ERROR: ")
		fmt.Println(err)
		return 1
	}

	logrus.Debug("Refreshing oauth token via URL ", u)

	v := url.Values{}
	v.Set("grant_type", "refresh_token")
	v.Set("client_id", c.Config.ClientCredentials.ClientId)
	v.Set("client_secret", c.Config.ClientCredentials.ClientSecret)
	v.Set("refresh_token", c.Config.ClientCredentials.RefreshToken)

	data, err := c.post(u, v)
	if err != nil {
		fmt.Println("ERROR: ")
		fmt.Println(err)
		return 1
	}

	t := OAuthResponse{}
	err = json.Unmarshal(*data, &t)

	if err != nil {
		fmt.Println("ERROR: ")
		fmt.Println(err)
		return 2
	}

	updateConfig(c.Config, &t)

	return 0
}

// Get a new token.
func (c *ClientCredentials) Get() int {
	logrus.Debug("Getting oauth token via client_credentials grant")

	u, err := buildUrl(c.URL, "token")
	if err != nil {
		fmt.Println("ERROR: ")
		fmt.Println(err)
		return 1
	}

	logrus.Debug("Getting oauth token via URL ", u)

	v := url.Values{}
	v.Set("grant_type", "client_credentials")
	v.Set("client_id", c.Config.ClientCredentials.ClientId)
	v.Set("client_secret", c.Config.ClientCredentials.ClientSecret)

	data, err := c.post(u, v)
	if err != nil {
		fmt.Println("ERROR: ")
		fmt.Println(err)
		return 1
	}

	t := OAuthResponse{}
	err = json.Unmarshal(*data, &t)

	if err != nil {
		fmt.Println("ERROR: ")
		fmt.Println(err)
		return 2
	}

	updateConfig(c.Config, &t)

	return 0
}

// GetClient creates and returns a new OAuth client.
func GetClient(c *config.Config) OAuth {
	return &ClientCredentials{c, uri}
}
