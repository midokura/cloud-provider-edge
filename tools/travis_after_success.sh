#! /bin/sh

set -e

if [ "$CI_TARGET" = "cover" ]; then
	exec $GOPATH/bin/goveralls -coverprofile=profile.cov -service=travis-ci
else
	exit 0
fi
