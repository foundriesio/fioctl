COMMIT?=$(shell git describe HEAD)$(shell git diff --quiet || echo '+dirty')

# Use linker flags to provide commit info
VERSION_LDFLAGS=-X=github.com/foundriesio/fioctl/subcommands/version.Commit=$(COMMIT)
COMMON_LDFLAGS=-v -s -w -linkmode=external $(VERSION_LDFLAGS)

linter:=$(shell which golangci-lint 2>/dev/null || echo $(HOME)/go/bin/golangci-lint)
builder:=$(shell which xgo 2>/dev/null || echo $(HOME)/go/bin/xgo)

# A crazy-max/xgo creates files as root. The below script sets proper ownership.
docker_command:=xgo-build . && chown --reference /build/bin /build/bin/fioctl-*

build: fioctl-linux-amd64 fioctl-linux-arm64 fioctl-windows-amd64 fioctl-darwin-amd64 fioctl-darwin-arm64
	@true

fioctl-static:
	CGO_ENABLED=0 go build -a -ldflags '-w -extldflags "-static"' -o ./bin/fioctl-static ./main.go

fioctl-linux-amd64:
fioctl-linux-arm64:
fioctl-linux-arm-7:
fioctl-windows-amd64:
fioctl-darwin-amd64:
fioctl-darwin-arm64:
fioctl-%:
	@test -x $(builder) || (echo "Please install xgo toolchain $(HOME)/go/bin: go install github.com/techknowlogick/xgo@v1.7.0+1.19.5")
	$(eval GOOS:=$(shell echo $* | cut -f1 -d\- ))
	$(eval GOARCH:=$(shell echo $* | cut -f2- -d\-))
	$(eval TARGET_GOTAGS:=netgo,osusergo)
	$(eval TARGET_GOTAGS_EXT:=netgo,osusergo,static_build)
	$(eval TARGET_GOTAGS:=$(if $(shell test $* = linux-amd64 && echo "ok"),$(TARGET_GOTAGS_EXT),$(TARGET_GOTAGS)))
	$(eval TARGET_LDFLAGS:=$(if $(shell test $(GOOS) = linux && echo "ok"),'-extldflags=-static -O1',))
	$(eval TARGET_LDFLAGS:=$(if $(shell test $(GOOS) = windows && echo "ok"),'-extldflags=-static',$(TARGET_LDFLAGS)))
	# static PIE is not yet supported on Arm by GCC
	$(eval TARGET_LDFLAGS:=$(if $(shell test $* = linux-amd64 && echo "ok"),-buildmode=pie '-extldflags=-static-pie -O1',$(TARGET_LDFLAGS)))
	$(eval COMBINED_LDFLAGS=$(COMMON_LDFLAGS) $(TARGET_LDFLAGS))
	@mkdir -p bin .tmpbin && echo '#!/bin/sh' > .tmpbin/cmd && echo '$(docker_command)' >> .tmpbin/cmd && chmod 755 .tmpbin/cmd
	$(builder) --targets=$(GOOS)/$(GOARCH) -out bin/fioctl --tags=$(TARGET_GOTAGS) --ldflags "$(COMBINED_LDFLAGS)" \
		--image="ghcr.io/crazy-max/xgo" --dockerargs "-v=$$(pwd)/.tmpbin:/tmpbin,--entrypoint=/tmpbin/cmd" .
	@rm .tmpbin/cmd && rm -r .tmpbin

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

