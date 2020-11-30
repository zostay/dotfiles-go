#!/bin/zsh

set -xe

docker build -t zostay/dotfiles-go .
docker push zostay/dotfiles-go
