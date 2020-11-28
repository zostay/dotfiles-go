#!/bin/sh

useradd sterling --home-dir /home/sterling --uid "$UID" --no-create-home --shell /bin/zsh
"$@"

