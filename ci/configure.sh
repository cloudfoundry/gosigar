#!/usr/bin/env bash

fly -t bosh-ecosystem set-pipeline \
    -p gosigar \
    -c ci/pipeline.yml
