A simple tool to interact with the Foundries.io REST API for managing a
Factory. Its based on the Foundries.io "ota-lite" API defined here:

 https://app.swaggerhub.com/apis/foundriesio/ota-lite/

## Using

You must first authenticate with the server before using this tool with:
~~~
  fioctl login
~~~

Most commands require a "factory" argument. This can be defaulted inside
`$HOME/.config/fioctl.yaml`
~~~
  factory: <The name of your factory>
~~~

You can then view your fleet of devices with `fioctl device list`, or
start to see the Targets(ie "builds") applicable to your devices with the
`fioctl targets list`.

The rest of the commands can be discovered by running `fioctl device --help`
and `fioctl targets --help`.

## Building
~~~
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
