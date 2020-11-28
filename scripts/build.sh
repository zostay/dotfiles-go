#!/bin/bash

set -xe

go generate
go build
go test
golangci-lint run
