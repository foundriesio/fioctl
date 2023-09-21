#!/bin/bash
if test -n "${XXX_INSIDE_CONTAINER}"; then
    # A crazy-max/xgo creates files as root. The below script sets proper ownership.
    xgo-build "$@" && chown --reference $(dirname /build/${OUT}) /build/${OUT}-*
    exit $?
fi

TARGET_PREFIX=bin/fioctl
TARGET_HSM=0
TARGET_STATIC=1
TARGET_LDFLAGS="'-extldflags=-O1'"
TARGET_GOTAGS=netgo,osusergo
GOOS=$(echo ${PLATFORM} | cut -f1 -d-)
GOARCH=$(echo ${PLATFORM} | cut -f2- -d-)

# Check if the binary should support HSM devices via PKCS11
if test "x${GOOS}" = "xhsm"; then
    GOOS=$(echo ${GOARCH} | cut -f1 -d-)
    GOARCH=$(echo ${GOARCH} | cut -f2- -d-)
    TARGET_HSM=1
    TARGET_STATIC=0
    TARGET_PREFIX=${TARGET_PREFIX}-hsm
fi

# For dynamically linked binaries this is just a fancy way to tell which Libc it requires.
# Currently, the XGo image we use only supports linking against the Libc.
if test "x${GOOS}" = "xlibc6"; then
    GOOS=$(echo ${GOARCH} | cut -f1 -d-)
    GOARCH=$(echo ${GOARCH} | cut -f2- -d-)
    TARGET_STATIC=0
    TARGET_PREFIX=${TARGET_PREFIX}-libc6
fi

if test "x${TARGET_STATIC}" = "x1"; then
    # Darwin officially disallows building static binaries.
    # Even internal linker always dynlinks libSystem.B.dylib, libresolv.9.dylib, CoreFoundation, and Security.
    test "x${GOOS}" = "xwindows" && TARGET_LDFLAGS="'-extldflags=-static -O1'"
    test "x${GOOS}" = "xlinux" && TARGET_LDFLAGS="'-extldflags=-static -O1'"
    if test "x${GOOS}-${GOARCH}" = "xlinux-amd64"; then
        # For Amd64 Linux add binary hardening using PIE and Full RELRO.
        # Static PIE is not supported on Arm Linux.
        TARGET_GOTAGS=${TARGET_GOTAGS},static_build
        TARGET_LDFLAGS="-buildmode=pie '-extldflags=-static-pie -O1'"
    fi
else
    test "x${TARGET_HSM}" = "x1" && TARGET_GOTAGS=${TARGET_GOTAGS},hsm
    # For Linux add binary hardening using PIE and Full RELRO.
    test "x${GOOS}" = "xlinux" && TARGET_LDFLAGS="-buildmode=pie '-extldflags=-pie -O1 -z relro -z now'"
fi

${BUILDER} \
    -image=${IMAGE} \
    -targets=${GOOS}/${GOARCH} \
    -out ${TARGET_PREFIX} \
    -tags=${TARGET_GOTAGS} \
    -ldflags="-v -s -w -linkmode=external ${TARGET_LDFLAGS} ${EXTRA_LDFLAGS}" \
    -dockerargs "-v=$(pwd)/build.sh:/xxxbin/build,-e=XXX_INSIDE_CONTAINER=1,--entrypoint=/xxxbin/build" \
    .
