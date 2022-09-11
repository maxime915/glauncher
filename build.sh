#!/usr/bin/env bash

# Author: https://belief-driven-design.com/build-time-variables-in-go-51439b26ef9/

# STEP 1: Determinate the required values

PACKAGE="github.com/maxime915/glauncher"
VERSION="$(git describe --tags --always --abbrev=0 --match='v[0-9]*.[0-9]*.[0-9]*' 2> /dev/null | sed 's/^.//')"
COMMIT_HASH="$(git rev-parse --short HEAD)"
BUILD_TIMESTAMP=$(date '+%Y-%m-%dT%H:%M:%S')

# STEP 2: Build the ldflags

LDFLAGS=(
  "-X '${PACKAGE}/version.Version=${VERSION}'"
  "-X '${PACKAGE}/version.CommitHash=${COMMIT_HASH}'"
  "-X '${PACKAGE}/version.BuildTimestamp=${BUILD_TIMESTAMP}'"
)

# STEP 3: Actual Go build process

go build -ldflags="${LDFLAGS[*]}" cmd/f/f.go
go build -ldflags="${LDFLAGS[*]}" cmd/glauncher_cli/glauncher_cli.go
