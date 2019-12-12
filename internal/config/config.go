package config

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Token string `yaml:"token,omitempty"`

	ClientCredentials struct {
		ClientId     string  `yaml:"client_id,omitempty"`
		ClientSecret string  `yaml:"client_secret,omitempty"`
		TokenType    string  `yaml:"token_type,omitempty"`
		AccessToken  string  `yaml:"access_token,omitempty"`
		RefreshToken string  `yaml:"refresh_token,omitempty"`
		ExpiresIn    float64 `yaml:"expires_in,omitempty"`
		Created      string  `yaml:"created,omitempty"`
	}
}

// Save the config data to file.
func (c *Config) Save(f string) {
	logrus.Debug("Saving config file")

	data, err := yaml.Marshal(c)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = ioutil.WriteFile(f, data, 0600)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// ReadClientCredentials read client id and secret from stdin.
func (c *Config) ReadClientCredentials() {
	logrus.Debug("Reading client ID/secret from stdin")

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Printf("Please provide the client ID and client secret\n\n")
	fmt.Print("Client ID: ")
	scanner.Scan()
	clientId := scanner.Text()

	fmt.Print("Client secret: ")
	scanner.Scan()
	clientSecret := scanner.Text()

	if clientId == "" || clientSecret == "" {
		fmt.Println("Client ID and client credentials are both required.")
		os.Exit(1)
	}

	c.ClientCredentials.ClientId = clientId
	c.ClientCredentials.ClientSecret = clientSecret
}

// Load the provided YAML config file.
func Load(f string) *Config {
	logrus.Debug("Loading yaml config file", f)
	var c Config

	// Suppress the error: in case it doesn't exist we return an empty struct.
	source, _ := ioutil.ReadFile(f)

	err := yaml.Unmarshal(source, &c)
	if err != nil {
		fmt.Println(err)
	}

	return &c
}
