COMMIT:=$(shell git describe HEAD)$(shell git diff --quiet || echo '+dirty')

# Use linker flags to provide commit info
LDFLAGS=-ldflags "-X=github.com/foundriesio/fioctl/subcommands/version.Commit=$(COMMIT)"

linter:=$(shell which golangci-lint 2>/dev/null || echo $(HOME)/go/bin/golangci-lint)

build: fioctl-linux-amd64 fioctl-linux-arm64 fioctl-windows-amd64 fioctl-darwin-amd64 fioctl-darwin-arm64
	@true

fioctl-static:
	CGO_ENABLED=0 go build -a -ldflags '-w -extldflags "-static"' -o ./bin/fioctl-static ./main.go

fioctl-linux-amd64:
fioctl-linux-arm64:
fioctl-linux-armv7:
fioctl-windows-amd64:
fioctl-darwin-amd64:
fioctl-darwin-arm64:
fioctl-%:
	CGO_ENABLED=0 \
	GOOS=$(shell echo $* | cut -f1 -d\- ) \
	GOARCH=$(shell echo $* | cut -f2 -d\-) \
		go build $(LDFLAGS) -o bin/$@ main.go
	@if [ "$@" = "fioctl-windows-amd64" ]; then mv bin/$@ bin/$@.exe; fi

format:
	@gofmt -l  -w ./
check:
	@test -z $(shell gofmt -l ./ | tee /dev/stderr) || echo "[WARN] Fix formatting issues with 'make format'"
	@test -x $(linter) || (echo "Please install linter from https://github.com/golangci/golangci-lint/releases/tag/v1.51.2 to $(HOME)/go/bin")
	$(linter) run

# Use the image for Dockerfile.build to build and install the tool.
container-init:
	docker build -t fioctl-build -f Dockerfile.build .

container-build:
	docker run --rm -ti -v $(shell pwd):/fioctl fioctl-build make build

