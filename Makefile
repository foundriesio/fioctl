COMMIT?=$(shell git describe HEAD)$(shell git diff --quiet || echo '+dirty')

# Use linker flags to provide commit info
VERSION_LDFLAGS=-X=github.com/foundriesio/fioctl/subcommands/version.Commit=$(COMMIT)
BUILD_STATIC=1
PREFIX=bin/fioctl

linter:=$(shell which golangci-lint 2>/dev/null || echo $(HOME)/go/bin/golangci-lint)
builder:=$(shell which xgo 2>/dev/null || echo $(HOME)/go/bin/xgo)
xgopkg::=src.techknowlogick.com/xgo@v1.7.0+1.19.5
xgoimg::=ghcr.io/crazy-max/xgo:1.21.0

# Platforms listed in build and build-hsm targets are fully supported and tested.
# See the README for a full list of platforms tentatively supported by our XGo cross-compiler toolchain.

# Statically linked binary without HSM support on platforms that support static binaries.
# On platforms not supporting static binaries (e.g. Darwin) this builds dynamic library without HSM support.
build: fioctl-linux-amd64 fioctl-linux-arm64 fioctl-windows-amd64 fioctl-darwin-amd64 fioctl-darwin-arm64
	@true

# Dynamically linked binary with HSM support.
# There is no way to build static binary with HSM support due to the PKCS11 design choice to use dlopen.
# For Darwin and Windows we link against the lowest possible library version for the greatest compatibility.
# For Linux we link against GNU Libc 6; so it must be installed on a target system.
build-hsm: fioctl-hsm-darwin-amd64 fioctl-hsm-darwin-arm64 fioctl-hsm-windows-amd64 fioctl-hsm-libc6-linux-amd64
	@true

# Below 3 targets allow you to build the local binary using your host toolchain
fioctl-local-dynlink:
	go build -ldflags '-s -w' -o ./bin/fioctl-local-dynlink ./main.go

fioctl-hsm-local-dynlink:
	go build -tags=hsm -ldflags '-s -w' -o ./bin/fioctl-hsm-local-dynlink ./main.go

fioctl-local-static:
	go build -tags=netgo,osusergo -ldflags '-s -w -extldflags=-static' -o ./bin/fioctl-local-static ./main.go

fioctl-%: has-builder
	@echo "Bulding for $*"; \
		PLATFORM=$* \
		IMAGE=$(xgoimg) \
		BUILDER=$(builder) \
		EXTRA_LDFLAGS=$(VERSION_LDFLAGS) \
		./build.sh

has-builder:
	@test -x $(builder) || (echo 'Please install xgo toolchain using "make install-builder"' && exit 1)

install-builder:
	@go install $(xgopkg)
	@docker pull $(xgoimg)

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

