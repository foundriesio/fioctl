# Fioctl TUF metadata

This is a special branch of Fioctl that includes its TUF metadata for releases
published to Github.

This repository is managed using [go-tuf](https://github.com/theupdateframework/go-tuf/releases/download/v0.6.0/tuf_0.6.0_linux_amd64.tar.gz)
and a combination of Github actions and release scripts.
You can also install it using `go install github.com/theupdateframework/go-tuf/cmd/tuf@v0.6.0`

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
$ git checkout tuf-metadata
$ tuf gen-key --expires 365 root
```

Next send a copy of `staged/root.json` to next person with root key:
```
# copy the new root.json into ./staged
$ tuf sign root.json
```

One way to do that is to `git commit` and `git push` your changes to `staged/root.json`.

This file would be copied to the next person in the chain if needed.
Once the signature threshold is reached,
the final person will do a `tuf commit` and then do a git commit/push to update the new root.json.

## Adding a new targets key:
In order to be able to create new releases, you need to own 3 keys: targets, snapshot, timestamp:
```
$ git checkout tuf-metadata
$ tuf gen-key --expires 365 targets
$ tuf gen-key --expires 365 snapshot
$ tuf gen-key --expires 365 timestamp
```

After doing that, send a copy of `staged/root.json` for signature to a person owning a root key.
If you own a root key, the above commands will add your signature, so you can commit the changes.

## Adding a new $X
The same as the root key procedure.
