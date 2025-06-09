#!/bin/bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

setup=false
bench=false
plot=false

for cmd in "$@"; do
    case $cmd in
        setup) setup=true ;;
        bench) bench=true ;;
        plot) plot=true ;;
        *) 
            echo "Unknown command: $cmd"
            ;;
    esac
done

export PLOTS_DIR="${SCRIPT_DIR}/plots"

if [[ "$setup" == "true" ]]; then
    echo "---"
    echo "Setting up Akesod"
    cd ../akesod/
    bash setup-akesod.sh
fi

if [[ "$bench" == "true" ]]; then
    echo "---"
    echo "Benchmarking Figure 9 - Time to re-encrypt a bucket of varying sizes where each object is 2MB"
    cd ../akesod/evaluations-bktSizeVsLatency
    bash automate.sh 10
    echo "Completed"
    echo "---"
    
    echo "Benchmarking Figure 10 - Time to re-encrypt a 1G bucket, varying object sizes in the bucket"
    cd ../akesod/evaluations-fixedBkt-varyingFile
    bash automate.sh 10
    echo "Completed"
    echo "---"
    echo "Done benchmarking read/write latency"
fi

export PLOTS_DIR="${SCRIPT_DIR}/plots"
if [[ "$plot" == "true" ]]; then
    echo "---"
    echo "Copying data"

    echo "---"
    echo "Transform benchmark data for plotting"

    cd $PLOTS_DIR
    echo "---"
    echo "Creating fig9 with gnuplot"
    gnuplot fig9.gpi

    echo "---"
    echo "Creating fig10 with gnuplot"
    gnuplot fig10.gpi
fi
