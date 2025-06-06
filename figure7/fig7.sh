#!/bin/bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

bench=false
plot=false

for cmd in "$@"; do
    case $cmd in
        bench) bench=true ;;
        plot) plot=true ;;
        *) 
            echo "Unknown command: $cmd"
            ;;
    esac
done

export CMEK_DIR="${SCRIPT_DIR}/cmek"
export PLOTS_DIR="${SCRIPT_DIR}/plots"

if [[ "$bench" == "true" ]]; then
    echo "---"
    echo "Benchmarking read/write latency"


    echo "---"
    echo "Done benchmarking read/write latency"
fi

if [[ "$plot" == "true" ]]; then
    echo "---"
    echo "Copying data"

    echo "---"
    echo "Transform benchmark data for plotting"

    echo "---"
    echo "Creating fig7 with gnuplot"
    # gnuplot fig7.gpi
fi
