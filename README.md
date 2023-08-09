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

## Adding a new root key:
```
$ git clone https://github.com/foundriesio/fioctl
$ git checkout -b tuf-metadata
$ tuf gen-key --expires 365 root
```

Next send a copy of `staged/root.json` to next person with root key:
```
# copy the new root.json into ./staged
$ tuf sign root.json
```

This file would be copied to the next person in the chain if needed. One the
signature threshold is reached, the final person will do a `tuf commit` and
then do a git commit/push to update the new root.json.

## Adding a new $X
The same as the root key procedure.
