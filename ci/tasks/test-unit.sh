#!/usr/bin/env bash

set -e

export GOPATH=$PWD/gopath
export PATH=$PATH:$GOPATH/bin
cd gopath/src/github.com/cloudfoundry/gosigar
bin/test-unit
