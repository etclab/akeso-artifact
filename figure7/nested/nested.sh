#!/bin/bash

set -e

PREFIX="atp"
STRATEGY="nested"

REGION="us-east4"
ZONE="us-east4-b"

PROJECT="ornate-flame-397517"

PWD=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

BUCKET_NAME=${PREFIX}-${STRATEGY}
VM_NAME=${BUCKET_NAME}-vm

setup_bucket=false
setup_vm=false
setup_nested=false
benchmark=false

REPO="gcsfuse-nested"

for cmd in "$@"; do
    case $cmd in
        setup_bucket) setup_bucket=true ;;
        setup_vm) setup_vm=true ;;
        setup_nested) setup_nested=true ;;
        benchmark) benchmark=true ;;
        *) 
            echo "Unknown command: $cmd"
            ;;
    esac
done

if [[ "$setup_bucket" == "true" ]]; then
    echo "Creating bucket"

    gcloud storage buckets create gs://${BUCKET_NAME} --location=${REGION} \
        --public-access-prevention --soft-delete-duration=0 \
        --uniform-bucket-level-access
fi

if [[ "$setup_vm" == "true" ]]; then
    echo "Setup vm"

    gcloud compute instances create ${VM_NAME} \
        --image-family=ubuntu-2404-lts-amd64 --image-project=ubuntu-os-cloud \
        --zone=${ZONE} --machine-type=n2d-standard-2 --boot-disk-size=20GB
fi

if [[ "$setup_nested" == "true" ]]; then

    echo "Cloning gcsfuse repo"
    cd ${PWD}

    if [[ ! -d ${REPO} ]]; then
        git clone https://github.com/etclab/gcsfuse.git ${REPO}
        cd ${REPO}
        git pull origin akeso-nested
        git checkout -b akeso-nested
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
    # python3 run-bench.py <sets> <reps>
    python3 run-bench.py 1 5
    cd -
fi

