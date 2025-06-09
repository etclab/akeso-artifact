#!/bin/bash

set -eo pipefail

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

if [[ "$setup" == "true" ]]; then
    echo "---"
    echo "Setting up Akesod"
    cd ../akesod/
    bash setup-akesod.sh
fi

if [[ "$bench" == "true" ]]; then
    echo "Assuming setup is complete"
    # Check if variables are already set, if not prompt for them
    if [[ -z "$PROJECT_ID" ]]; then
        read -p "Enter PROJECT_ID: " PROJECT_ID
        export PROJECT_ID
    fi

    if [[ -z "$REGION" ]]; then
        read -p "Enter REGION: " REGION
        export REGION
    fi

    if [[ -z "$CLOUD_FUNCTION" ]]; then
        read -p "Enter CLOUD_FUNCTION: " CLOUD_FUNCTION
        export CLOUD_FUNCTION
    fi

    if [[ -z "$METADATAUPDATE_TOPIC" ]]; then
        read -p "Enter METADATAUPDATE_TOPIC: " METADATAUPDATE_TOPIC
        export METADATAUPDATE_TOPIC
    fi

    # Display the set values for confirmation
    echo "Configuration:"
    echo "  PROJECT_ID: $PROJECT_ID"
    echo "  REGION: $REGION"
    echo "  CLOUD_FUNCTION: $CLOUD_FUNCTION"
    echo "  METADATAUPDATE_TOPIC: $METADATAUPDATE_TOPIC"
    echo ""
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
export AKESOD_DIR="${SCRIPT_DIR}/../akesod/"
if [[ "$plot" == "true" ]]; then
    echo "---"
    echo "Copying data"

    echo "---"
    echo "Transform benchmark data for plotting"
    cd $AKESOD_DIR

    export VENV_DIR="venv"

    # Create a virtual environment if it doesn't exist
    if [ ! -d "$VENV_DIR" ]; then
        echo "Creating Python virtual environment in './$VENV_DIR/'..."
        python3 -m venv "$VENV_DIR"
    fi

    # Activate the virtual environment and run commands within it
    source "$VENV_DIR/bin/activate"

    # Check for numpy and install if not present (now safely inside the venv)
    if ! python -c "import numpy" &> /dev/null; then
        echo "numpy not found. Installing into the virtual environment..."
        pip install numpy
    fi

    echo "Processing the data for fig 9"
    cd evaluations-bktSizeVsLatency

    # Check if new experimental results are generated, if not copy old results to be processed
    files=(update*.dat)
    if (( ${#files[@]} == 1 )); then
        cp old_data/*.dat .
    fi

    
    echo "Extracting the data from multiple runs"
    python process-data.py
    mv combined.dat $PLOTS_DIR/time-reencrypt-bucket-hist-vary-bucket-size-w-err.dat
    mv cmek_means.gpi $PLOTS_DIR/cmek_means.gpi
    rm update*.dat

    echo "Processing the data for fig 10"
    cd $AKESOD_DIR/evaluations-fixedBkt-varyingFile
    
    # Check if new experimental results are generated, if not copy old results to be processed
    files=(update*.dat)
    if (( ${#files[@]} == 1 )); then
        cp old_data/*.dat .
    fi

    echo "Extracting the data from multiple runs"
    python process-data.py
    mv combined.dat $PLOTS_DIR/time-reencrypt-bucket-hist-vary-object-size-w-err.dat
    rm update*.dat

    cd $PLOTS_DIR
    echo "---"
    echo "Creating fig9 with gnuplot"
    gnuplot fig9.gpi

    
    echo "---"
    echo "Creating fig10 with gnuplot"
    gnuplot fig10.gpi
fi
