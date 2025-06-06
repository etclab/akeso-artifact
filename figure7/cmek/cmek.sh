#!/bin/bash

set -e

PREFIX="pets25"
STRATEGY="cmek"

REGION="us-east1"
ZONE="us-east1-b"

# TODO: change project id or use env var with restricted service account
PROJECT="ornate-flame-397517"

PWD=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

BUCKET_NAME=${PREFIX}-${STRATEGY}
VM_NAME=${BUCKET_NAME}-vm
SOFT_KEY="projects/${PROJECT}/locations/${REGION}/keyRings/atp-keyring/cryptoKeys/soft-key"

setup_bucket=false
setup_vm=false
setup_cmek=false
benchmark=false

REPO="gcsfuse-cmek"

for cmd in "$@"; do
    case $cmd in
        setup_bucket) setup_bucket=true ;; # TODO: no need to setup bucket
        setup_vm) setup_vm=true ;; # TODO: as a first step, no need to setup vm too, run from local machine/server
        setup_cmek) setup_cmek=true ;; # TODO: need to setup cmek repo
        benchmark) benchmark=true ;; # TODO: ensure the service account allows access for mounting buckets
        *) 
            echo "Unknown command: $cmd"
            ;;
    esac
done

if [[ "$setup_bucket" == "true" ]]; then
    echo "Creating bucket"

    gcloud storage buckets create gs://${BUCKET_NAME} --location=${REGION} \
        --public-access-prevention --soft-delete-duration=0 \
        --uniform-bucket-level-access \
        --default-encryption-key=${SOFT_KEY}
fi

if [[ "$setup_vm" == "true" ]]; then
    echo "Setup vm"

    gcloud compute instances create ${VM_NAME} \
        --image-family=ubuntu-2404-lts-amd64 --image-project=ubuntu-os-cloud \
        --zone=${ZONE} --machine-type=n2d-standard-2 --boot-disk-size=20GB
fi

if [[ "$setup_cmek" == "true" ]]; then

    echo "Cloning gcsfuse repo"
    cd ${PWD}

    if [[ ! -d ${REPO} ]]; then
        # TODO: ensure the repository and commits are correct
        git clone https://github.com/etclab/gcsfuse.git ${REPO}
        cd ${REPO}
        git pull origin cmek
        git checkout -b cmek
        cd -

        echo "Setting up folders and configs"

        mkdir -p ${REPO}/smh/logs
        mkdir -p ${REPO}/smh/run
        mkdir -p ${REPO}/mnt

        # no need for cache/logging/debug during perf
        cat new-config.yaml > ${REPO}/smh/conf/config.yaml
        cat new-mount.sh > ${REPO}/smh/mount.sh

        cp run-bench.py ${REPO}
    fi

    cd ${REPO}
    ./smh/make.sh
    cd -
fi

if [[ "$benchmark" == "true" ]]; then
    echo "Running benchmark"

    cd ${PWD}
    cd ${REPO}
    # run 1 set with 50 reps each
    python3 run-bench.py 1
    cd -
fi

