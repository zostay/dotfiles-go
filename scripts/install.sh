#!/bin/bash

set -xe

./scripts/build.sh
go install ./...
