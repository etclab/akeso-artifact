#!/bin/bash

# this script downloads Go 1.24.3 to SCRIPT_DIR and adds to the PATH
# also replaces any existing Go installation so use with caution

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
ARCHIVE=go1.24.3.linux-amd64.tar.gz

cd $SCRIPT_DIR

wget https://go.dev/dl/${ARCHIVE}

sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.24.3.linux-amd64.tar.gz

echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

rm -rf $ARCHIVE

cd -