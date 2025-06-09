#!/bin/sh

rm -rf ./smh/run/filecache

./gcsfuse \
    --config-file smh/conf/config.yaml \
    --implicit-dirs \
    atp-nested mnt