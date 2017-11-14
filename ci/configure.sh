#!/usr/bin/env bash

fly -t production set-pipeline \
    -p gosigar \
    -c ci/pipeline.yml
