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

export PLOTS_DIR="${SCRIPT_DIR}/plots"

if [[ "$bench" == "true" ]]; then
    echo "---"
    echo "Ensure benchmarking read/write latency for fig7 exists"
fi

if [[ "$plot" == "true" ]]; then
    echo "---"
    echo "Transform benchmark data for plotting"
    python3 cdf-transform.py

    cd $PLOTS_DIR

    echo "---"
    echo "Creating fig8 with gnuplot"
    gnuplot fig8.gpi
fi
