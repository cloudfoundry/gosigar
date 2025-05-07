#!/usr/bin/env bash

fly -t "${CONCOURSE_TARGET:-bosh-ecosystem}" set-pipeline \
    -p gosigar \
    -c ci/pipeline.yml
