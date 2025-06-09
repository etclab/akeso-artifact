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
    echo "Benchmarking read/write latency"
    
    $SCRIPT_DIR/cmek/cmek.sh setup_cmek benchmark
    $SCRIPT_DIR/cmek-hsm/cmek-hsm.sh setup_cmek_hsm benchmark
    $SCRIPT_DIR/csek/csek.sh setup_csek benchmark
    $SCRIPT_DIR/keywrap/keywrap.sh setup_keywrap benchmark
    $SCRIPT_DIR/nested/nested.sh setup_nested benchmark
    $SCRIPT_DIR/strawman/strawman.sh setup_strawman benchmark

    echo "---"
    echo "Done benchmarking read/write latency"
fi

if [[ "$plot" == "true" ]]; then
    echo "---"
    echo "Transform benchmark data for plotting"
    python3 format-all-rel-tm.py 

    cd $PLOTS_DIR

    echo "---"
    echo "Creating fig7 with gnuplot"
    gnuplot fig7.gpi
fi
