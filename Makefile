COMMIT?=$(shell git describe HEAD)$(shell git diff --quiet || echo '+dirty')

# Use linker flags to provide commit info
LDFLAGS=-ldflags "-X=github.com/foundriesio/fioctl/subcommands/version.Commit=$(COMMIT)"

linter:=$(shell which golangci-lint 2>/dev/null || echo $(HOME)/go/bin/golangci-lint)

build: fioctl-linux-amd64 fioctl-linux-arm64 fioctl-windows-amd64 fioctl-darwin-amd64 fioctl-darwin-arm64
	@true

# Allows building a dyn-linked fioctl on platforms without pkcs11-tool (not built by default)
fioctl-cgo-pkcs11:
	CGO_ENABLED=1 go build -tags cgopki $(LDFLAGS) -o bin/$@ ./main.go

# Any target of "fioctl-{GOOS}-{GOARCH}" can be built for your desired platform.
# Below is a list of platforms supported and verified by Foundries.io.
# For a full list of potentially supported (by Golang compiler) see https://go.dev/doc/install/source#environment.
fioctl-linux-amd64:
fioctl-linux-arm64:
fioctl-windows-amd64:
fioctl-darwin-amd64:
fioctl-darwin-arm64:
fioctl-%:
	CGO_ENABLED=0 \
	GOOS=$(shell echo $* | cut -f1 -d\- ) \
	GOARCH=$(shell echo $* | cut -f2 -d\-) \
		go build $(LDFLAGS) -o bin/$@ main.go
	@if [ "$@" = "fioctl-windows-amd64" ]; then mv bin/$@ bin/$@.exe; fi

install-linter:
	echo "[WARN] Installing golangci binary version v1.51.2 at $(HOME)/go/bin/golangci-lint"
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell dirname $(linter)) v1.51.2

has-linter:
	@test -x $(linter) || (echo '[ERROR] Please install go linter using "make install-linter"' && exit 1)

linter-check: has-linter
	$(linter) run ${EXTRA_LINTER_FLAGS}
	$(linter) run --build-tags bashpki ${EXTRA_LINTER_FLAGS}
	$(linter) run --build-tags cgopki ${EXTRA_LINTER_FLAGS}
	$(linter) run --build-tags testhsm ${EXTRA_LINTER_FLAGS}

linter: has-linter
	$(linter) run --fix ${EXTRA_LINTER_FLAGS}

format-check:
	@test -z $(shell gofmt -l -s ./ | tee /dev/stderr) || echo "[WARN] Fix formatting issues with 'make format-check'"

format:
	@gofmt -l -s -w ./

check: format-check linter-check
	@true

install-test-pki-deps:
	apt install openssl softhsm2 opensc libengine-pkcs11-openssl

# This needs the following packages on Ubuntu: openssl softhsm2 opensc libengine-pkcs11-openssl
test-pki:
	go test ./x509/... -v -tags testhsm
	go test ./x509/... -v -tags testhsm,bashpki
	go test ./x509/... -v -tags testhsm,cgopki
