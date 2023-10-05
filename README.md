# Fioctl

A simple tool to interact with the Foundries.io REST API for managing a
Factory. Its based on the Foundries.io "ota-lite" API defined here:

[https://api.foundries.io/ota/](https://api.foundries.io/ota/)

## Using

You must first authenticate with the server before using this tool with:

~~~sh
fioctl login
~~~

Most commands require a "factory" argument. This can be defaulted inside
`$HOME/.config/fioctl.yaml`

~~~yaml
factory: <The name of your factory>
~~~

You can then view your fleet of devices with `fioctl device list`, or
start to see the Targets(ie "builds") applicable to your devices with the
`fioctl targets list`.

The rest of the commands can be discovered by running `fioctl device --help`
and `fioctl targets --help`.

## Building

~~~sh
make build  # builds all targets

# or build for your specific target
make fioctl-linux-amd64
make fioctl-darwin-amd64
make fioctl-windows-amd64

make container-init && make container-build && \
export PATH=$PATH:`pwd`/bin
~~~

## Making Changes

After making changes be sure to run `make format` which will run the go-fmt
tool against the source code.

## HSM support

The HSM support (for some commands) is provided via the OpenSC pkcs11-tool application,
which needs to be installed separately.
It is not needed if you do not plan to use HSM devices.
