#!/bin/sh -e

tuf_bin=/tmp/tuf
curl -L https://github.com/theupdateframework/go-tuf/releases/download/v0.6.0/tuf_0.6.0_linux_amd64.tar.gz | tar -C /tmp -xz tuf > ${tuf_bin}
if [ "$(md5sum ${tuf_bin})"  != "27fed62ea9c176b7866828be3e0fd42b  ${tuf_bin}" ] ; then
	echo "Invalid tuf binary"
	exit 1
fi
chmod +x ${tuf_bin}

git config user.name github-actions
git config user.email github-actions@github.com

if ! ${tuf_bin} status --valid-at "`date --rfc-3339=seconds -d '+48 hour' | sed 's/ /T/'`" timestamp ; then
		echo "refreshing timestamp metadata"
		${tuf_bin} timestamp --expires=7
		${tuf_bin} commit
fi

git add repository/*
if [ -z "$(git status --porcelain)" ] ; then
	echo "metadata does not need refreshing"
else
	echo "committing changes to metadata"
	git commit -m "updated by github action"
	git push
fi
