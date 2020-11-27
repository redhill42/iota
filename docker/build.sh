#!/bin/bash
set -e

cd ..
CROSS=linux/amd64 make binary
mkdir -p docker/bin
cp bundles/latest/binary/linux/amd64/* docker/bin
cd docker
docker build -t icloudway/iota .
rm -rf bin

