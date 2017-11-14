#!/usr/bin/env bash

set -e

export PATH=/usr/local/ruby/bin:/usr/local/go/bin:$PATH
export GOPATH=$(pwd)/gopath
cd gopath/src/github.com/cloudfoundry/gosigar
bin/test-unit
