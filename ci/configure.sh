#!/usr/bin/env bash

fly -t production set-pipeline \
    -p gosigar \
    -c pipeline.yml
