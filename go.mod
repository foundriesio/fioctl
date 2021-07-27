module github.com/foundriesio/fioctl

go 1.13

require (
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412 // indirect
	github.com/cheynewallace/tabby v1.1.0
	github.com/docker/go v1.5.1-1
	github.com/ethereum/go-ethereum v1.9.25
	github.com/fatih/color v1.9.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pelletier/go-toml v1.2.0
	github.com/shurcooL/go v0.0.0-20200502201357-93f07166e636
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.3.2
	github.com/theupdateframework/notary v0.6.1
	gopkg.in/yaml.v2 v2.3.0
)

replace github.com/docker/go => github.com/foundriesio/go v1.5.1-1.0.20210202214252-a487d04e824d
