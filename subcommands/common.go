package subcommands

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/cheynewallace/tabby"
	canonical "github.com/docker/go/canonical/json"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/shurcooL/go/indentwriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"

	"github.com/foundriesio/fioctl/client"
	"github.com/foundriesio/fioctl/subcommands/version"
)

var (
	Config client.Config
)

func RequireFactory(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("factory", "f", "", "Factory to list targets for")
	cmd.PersistentFlags().StringP("token", "t", "", "API token from https://app.foundries.io/settings/tokens/")
}

func Login(cmd *cobra.Command) *client.Api {
	ca := os.Getenv("CACERT")
	DieNotNil(viper.BindPFlags(cmd.Flags()))
	Config.Token = viper.GetString("token")
	url := os.Getenv("API_URL")
	if len(url) == 0 {
		url = "https://api.foundries.io"
	}
	if viper.GetBool("server.insecure_skip_verify") {
		Config.InsecureSkipVerify = true
	}
	if len(Config.Token) > 0 {
		if cmd.Flags().Lookup("factory") != nil && len(viper.GetString("factory")) == 0 {
			DieNotNil(fmt.Errorf("Required flag \"factory\" not set"))
		}
		return client.NewApiClient(url, Config, ca, version.Commit)
	}

	if len(Config.ClientCredentials.ClientId) == 0 {
		DieNotNil(fmt.Errorf("Please run: \"fioctl login\" first"))
	}
	if cmd.Flags().Lookup("factory") != nil && len(viper.GetString("factory")) == 0 {
		DieNotNil(fmt.Errorf("Required flag \"factory\" not set"))
	}
	creds := client.NewClientCredentials(Config.ClientCredentials)

	expired, err := creds.IsExpired()
	DieNotNil(err)

	if !expired && len(creds.Config.AccessToken) > 0 {
		return client.NewApiClient(url, Config, ca, version.Commit)
	}

	if len(creds.Config.AccessToken) == 0 {
		DieNotNil(creds.Get())
	} else if creds.HasRefreshToken() {
		DieNotNil(creds.Refresh())
	} else {
		DieNotNil(fmt.Errorf("Missing refresh token"))
	}
	SaveOauthConfig(creds.Config)
	Config.ClientCredentials = creds.Config
	return client.NewApiClient(url, Config, ca, version.Commit)
}

func SaveOauthConfig(c client.OAuthConfig) {
	viper.Set("clientcredentials.client_id", c.ClientId)
	viper.Set("clientcredentials.client_secret", c.ClientSecret)

	viper.Set("clientcredentials.access_token", c.AccessToken)
	viper.Set("clientcredentials.refresh_token", c.RefreshToken)
	viper.Set("clientcredentials.token_type", c.TokenType)
	viper.Set("clientcredentials.expires_in", c.ExpiresIn)
	viper.Set("clientcredentials.created", c.Created)

	// viper.WriteConfig isn't so great for this. It doesn't just write
	// these values but any other flags that were present when this runs.
	// This gets run automatically when "logging in". So you sometimes
	// accidentally write CLI flags viper finds to the file, that you
	// don't intend to be saved. So we do it the hard way:
	name := viper.ConfigFileUsed()
	if len(name) == 0 {
		logrus.Debug("Guessing config file from path")
		path, err := homedir.Expand("~/.config")
		DieNotNil(err)
		name = filepath.Join(path, "fioctl.yaml")
	}
	// Try to read in config
	cfg := make(map[string]interface{})
	buf, err := os.ReadFile(name)
	if err == nil {
		DieNotNil(yaml.Unmarshal(buf, &cfg), "Unable unmarshal configuration:")
	}
	val := viper.Get("clientcredentials")
	cfg["clientcredentials"] = val
	if len(c.DefaultOrg) > 0 {
		cfg["factory"] = c.DefaultOrg
	}
	buf, err = yaml.Marshal(cfg)
	DieNotNil(err, "Unable to marshall oauth config:")
	DieNotNil(os.WriteFile(name, buf, os.FileMode(0644)), "Unable to update config: ")
}

// An os.Exit exits immediately, skipping all deferred functions
// We need a way to execute the finalizing code in some cases.
type LastWill = func()

var onLastWill []LastWill

func AddLastWill(lastWill LastWill) {
	onLastWill = append(onLastWill, lastWill)
}

func DieNotNil(err error, message ...string) {
	if err != nil {
		parts := []interface{}{"ERROR:"}
		for _, p := range message {
			parts = append(parts, p)
		}
		parts = append(parts, err)
		fmt.Println(parts...)
		for _, w := range onLastWill {
			w()
		}
		os.Exit(1)
	}
}

func Tabby(indent int, columns ...interface{}) *tabby.Tabby {
	var out io.Writer = os.Stdout
	if indent > 0 {
		out = indentwriter.New(out, indent)
	}
	tab := tabby.NewCustom(tabwriter.NewWriter(out, 0, 0, 2, ' ', 0))
	if len(columns) > 0 {
		tab.AddHeader(columns...)
	}
	return tab
}

// Copied from canonical.MarshalIndent, but replaced the Marshal call with MarshalCanonical.
func MarshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	b, err := canonical.MarshalCanonical(v)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = canonical.Indent(&buf, b, prefix, indent)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func AssertWritable(path string) {
	st, err := os.Stat(path)
	DieNotNil(err)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, st.Mode())
	if err != nil {
		fmt.Println("ERROR: File is not writeable:", path)
		os.Exit(1)
	}
	f.Close()
}

func IsSliceSetEqual[T comparable](first, second []T) bool {
	// Verify that two slices are equal as sets in just 3 O(1) iterations.
	// firstMap[key] = false	<- key present in first;
	// firstMap[key] = true		<- key present in both first and second.
	firstMap := make(map[T]bool, len(first))
	for _, val := range first {
		firstMap[val] = false
	}
	for _, val := range second {
		if _, inFirst := firstMap[val]; !inFirst {
			return false
		}
		firstMap[val] = true
	}
	for _, inSecond := range firstMap {
		if !inSecond {
			return false
		}
	}
	return true
}

func SliceRemove[T comparable](in []T, item T) (out []T) {
	out = make([]T, 0, len(in))
	for _, elem := range in {
		if item != elem {
			out = append(out, elem)
		}
	}
	return
}

type MutuallyExclusiveFlags struct {
	flags map[string]*bool
}

func (f *MutuallyExclusiveFlags) Add(cmd *cobra.Command, flagName, helpText string) {
	if f.flags == nil {
		f.flags = make(map[string]*bool)
	}
	val := false
	f.flags[flagName] = &val
	cmd.Flags().BoolVarP(f.flags[flagName], flagName, "", false, helpText)
}

func (f *MutuallyExclusiveFlags) GetFlag() (string, error) {
	flagName := ""
	for k, v := range f.flags {
		if *v && len(flagName) == 0 {
			flagName = k
		} else if *v {
			return "", fmt.Errorf("--%s and --%s are mutually exclusive", flagName, k)
		}

	}
	return flagName, nil
}

// Find an entry in the PATH we can write to. For example, on MacOS git is
// installed under /usr/bin but even root can't write to that because of
// filesystem protection logic they have.
func FindWritableDirInPath(helperPath string) string {
	path := os.Getenv("PATH")
	paths := make(map[string]bool)
	for _, part := range filepath.SplitList(path) {
		paths[part] = true
	}

	// Give preference to helper executable location if its in PATH
	if len(helperPath) > 0 {
		if _, ok := paths[helperPath]; ok {
			if IsWritable(helperPath) {
				return helperPath
			}
		}
	}

	// Now try everything
	for _, path := range filepath.SplitList(path) {
		if IsWritable(path) {
			return path
		}
	}
	return ""
}

func ShowPages(showPage int, next *string) {
	if next != nil {
		fmt.Print("\nNext page can be viewed with: ")
		found := false
		for i := 0; i < len(os.Args); i++ {
			arg := os.Args[i]
			if (len(arg) > 2 && arg[:2] == "-p") || (len(arg) > 7 && arg[:7] == "--page=") {
				fmt.Printf("-p%d ", showPage+1)
				found = true
			} else if arg == "-p" || arg == "--page" {
				fmt.Printf("-p%d ", showPage+1)
				found = true
				i++
			} else {
				fmt.Print(os.Args[i], " ")
			}
		}
		if !found {
			fmt.Print("-p", showPage+1)
		}
		fmt.Println()
	}
}
