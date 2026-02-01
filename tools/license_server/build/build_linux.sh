#!/bin/bash
# Run this script on Linux Ubuntu server
set -e
echo "Building License Server for Linux..."
CGO_ENABLED=1 go build -o license_server_linux .
echo "Done: license_server_linux"
