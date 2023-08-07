# Fioctl TUF metadata

This is a special branch of Fioctl that includes its TUF metadata for releases
published to Github.

This repository is managed using [go-tuf](https://github.com/theupdateframework/go-tuf/releases/download/v0.6.0/tuf_0.6.0_linux_amd64.tar.gz)
and a combination of Github actions and release scripts.

## Creating a new release
```
TUF_SNAPSHOT_PASSPHRASE="TODO" \
TUF_TIMESTAMP_PASSPHRASE="TODO" \
TUF_TARGETS_PASSPHRASE="TODO" \
    ./add-release-targets v0.34
```
