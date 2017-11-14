#!/usr/bin/env bash

set -e

export GOPATH=$PWD/gopath
cd gopath/src/github.com/cloudfoundry/gosigar
bin/test-unit
