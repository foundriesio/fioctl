A simple tool to interact with the Foundries.io REST API for managing a
Factory.

## Building
~~~
 make build  # builds all targets

 # or build for your specific target
 make fioctl-linux-amd64
 make fioctl-darwin-amd64
 make fioctl-windows-amd64
~~~

## Making Changes
After making changes be sure to run `make format` which will run the go-fmt
tool against the source code.
