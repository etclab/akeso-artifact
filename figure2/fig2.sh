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

export ART_DIR="${SCRIPT_DIR}/art"
export DRAT_DIR="${SCRIPT_DIR}/drat-bench"
export PLOTS_DIR="${SCRIPT_DIR}/plots"

if [[ "$bench" == "true" ]]; then
    echo "---"
    echo "Benchmarking key update operation using ART"

    cd $ART_DIR
    make benchmark > art.bench
    cd -

    echo "---"
    echo "Benchmarking key update operation using Double Ratchet"

    cd $DRAT_DIR
    make drat-bench > drat.bench
    cd - 

    echo "---"
    echo "Done benchmarking key update operation"
fi

if [[ "$plot" == "true" ]]; then
    echo "---"
    echo "Copying data from ART and DRAT benchmarks"

    cp $ART_DIR/art.bench $PLOTS_DIR
    cp $DRAT_DIR/drat.bench $PLOTS_DIR

    echo "---"
    echo "Transform benchmark data for plotting"

    cd $PLOTS_DIR
    python3 transform.py

    echo "---"
    echo "Creating fig2 with gnuplot"
    gnuplot fig2.gpi
fi
