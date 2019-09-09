COMMIT:=$(shell git log -1 --pretty=format:%h)$(shell git diff --quiet || echo '_')

# Use linker flags to provide commit info
LDFLAGS=-ldflags "-X=github.com/foundriesio/fioctl/cmd.Commit=$(COMMIT)"

build: fioctl-linux-amd64 fioctl-windows-amd64 fioctl-darwin-amd64
	@true

fioctl-linux-amd64:
fioctl-linux-armv7:
fioctl-windows-amd64:
fioctl-darwin-amd64:
fioctl-%:
	GOOS=$(shell echo $* | cut -f1 -d\- ) \
	GOARCH=$(shell echo $* | cut -f2 -d\-) \
		go build $(LDFLAGS) -o bin/$@ main.go

format:
	@gofmt -l  -w ./
check:
	@test -z $(shell gofmt -l ./ | tee /dev/stderr) || echo "[WARN] Fix formatting issues with 'make fmt'"
