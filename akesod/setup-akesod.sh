#!/bin/bash

# Akeso Daemon Complete Setup Script
# This script automates the entire setup process for akesod

set -e  # Exit on any error

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    if ! command_exists gcloud; then
        print_error "gcloud CLI is not installed. Please install Google Cloud SDK first."
        exit 1
    fi
    
    if ! command_exists make; then
        print_error "make is not installed. Please install build tools."
        exit 1
    fi
    
    # Check if PROJECT_ID is set
    if [[ -z "${PROJECT_ID}" ]]; then
        print_error "PROJECT_ID environment variable is not set."
        echo "Please set it with: export PROJECT_ID=your-project-id"
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

# Setup directories and keys
setup_directories() {
    print_status "Setting up directories and key configuration..."
    
    # Create keys directory if it doesn't exist
    mkdir -p keys
    
    # Generate group member configuration
    # Note: This assumes the key files already exist. You may need to generate them first.
    cat > keys/4.conf << EOF
akesod  keys/akesod-ik-pub.pem   keys/akesod-ek-pub.pem
bob     keys/bob-ik-pub.pem      keys/bob-ek-pub.pem
cici    keys/cici-ik-pub.pem     keys/cici-ek-pub.pem
dave    keys/dave-ik-pub.pem     keys/dave-ek-pub.pem
EOF
    
    print_success "Key configuration created"
}

# Setup configuration file
setup_config() {
    print_status "Setting up configuration file..."
    
    if [[ ! -f "config/config.yaml.example" ]]; then
        print_warning "config/config.yaml.example not found. Please ensure it exists."
    else
        cp config/config.yaml.example config/config.yaml
        print_success "Configuration file copied"
        print_warning "Please review and update config/config.yaml with your specific values"
    fi
}

# Enable required Google Cloud services
enable_gcloud_services() {
    print_status "Enabling required Google Cloud services..."
    
    gcloud services enable \
        cloudkms.googleapis.com \
        cloudtasks.googleapis.com \
        cloudfunctions.googleapis.com \
        pubsub.googleapis.com \
        storage.googleapis.com \
        --project="$PROJECT_ID"
    
    print_success "Google Cloud services enabled"
}

# Create Pub/Sub topics
create_pubsub_topics() {
    print_status "Creating Pub/Sub topics..."
    
    # Create GroupSetup topic
    if gcloud pubsub topics describe GroupSetup --project="$PROJECT_ID" >/dev/null 2>&1; then
        print_warning "GroupSetup topic already exists"
    else
        gcloud pubsub topics create GroupSetup --project="$PROJECT_ID"
        print_success "GroupSetup topic created"
    fi
    
    # Create KeyUpdate topic
    if gcloud pubsub topics describe KeyUpdate --project="$PROJECT_ID" >/dev/null 2>&1; then
        print_warning "KeyUpdate topic already exists"
    else
        gcloud pubsub topics create KeyUpdate --project="$PROJECT_ID"
        print_success "KeyUpdate topic created"
    fi
    
    # Create MetadataUpdate topic
    if gcloud pubsub topics describe MetadataUpdate --project="$PROJECT_ID" >/dev/null 2>&1; then
        print_warning "MetadataUpdate topic already exists"
    else
        gcloud pubsub topics create MetadataUpdate --project="$PROJECT_ID"
        print_success "MetadataUpdate topic created"
    fi
}

# Setup Cloud KMS
setup_cloud_kms() {
    print_status "Setting up Cloud KMS..."
    
    # Create keyring
    if gcloud kms keyrings describe akeso_dev --location=us-east1 --project="$PROJECT_ID" >/dev/null 2>&1; then
        print_warning "Keyring akeso_dev already exists"
    else
        gcloud kms keyrings create akeso_dev --location=us-east1 --project="$PROJECT_ID"
        print_success "Keyring akeso_dev created"
    fi
    
    # Create software-protected key
    if gcloud kms keys describe key1 --keyring=akeso_dev --location=us-east1 --project="$PROJECT_ID" >/dev/null 2>&1; then
        print_warning "Key key1 already exists"
    else
        gcloud kms keys create key1 \
            --keyring=akeso_dev \
            --location=us-east1 \
            --purpose="encryption" \
            --protection-level="software" \
            --project="$PROJECT_ID"
        print_success "Software-protected key1 created"
    fi
    
    # Create HSM-protected key
    if gcloud kms keys describe key2 --keyring=akeso_dev --location=us-east1 --project="$PROJECT_ID" >/dev/null 2>&1; then
        print_warning "Key key2 already exists"
    else
        gcloud kms keys create key2 \
            --keyring=akeso_dev \
            --location=us-east1 \
            --purpose="encryption" \
            --protection-level="hsm" \
            --project="$PROJECT_ID"
        print_success "HSM-protected key2 created"
    fi
}

# Setup IAM permissions
setup_iam_permissions() {
    print_status "Setting up IAM permissions..."
    
    # Get project number
    export PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format="value(projectNumber)")
    print_status "Project number: $PROJECT_NUMBER"
    
    # Grant KMS permissions for key1
    gcloud kms keys add-iam-policy-binding key1 \
        --location=us-east1 \
        --keyring=akeso_dev \
        --member="serviceAccount:service-$PROJECT_NUMBER@gs-project-accounts.iam.gserviceaccount.com" \
        --role="roles/cloudkms.cryptoKeyEncrypterDecrypter" \
        --project="$PROJECT_ID"
    
    # Grant KMS permissions for key2
    gcloud kms keys add-iam-policy-binding key2 \
        --location=us-east1 \
        --keyring=akeso_dev \
        --member="serviceAccount:service-$PROJECT_NUMBER@gs-project-accounts.iam.gserviceaccount.com" \
        --role="roles/cloudkms.cryptoKeyEncrypterDecrypter" \
        --project="$PROJECT_ID"
    
    # Grant Pub/Sub publisher role to service agent
    gcloud projects add-iam-policy-binding "$PROJECT_ID" \
        --member="serviceAccount:service-${PROJECT_NUMBER}@gs-project-accounts.iam.gserviceaccount.com" \
        --role="roles/pubsub.publisher"
    
    print_success "IAM permissions configured"
}

# Build the application
build_application() {
    print_status "Building akesod application..."
    
    if [[ ! -f "Makefile" ]]; then
        print_error "Makefile not found. Please ensure you're in the correct directory."
        exit 1
    fi
    
    make
    print_success "Application built successfully"
}

# Deploy cloud function (optional - requires source code)
deploy_cloud_function() {
    print_status "Deploying encrypt-object cloud function..."
    
    if [[ -d "./cmd/gcs-utils/cloud-functions/encrypt-object/" ]]; then
        gcloud functions deploy encrypt-object \
            --gen2 \
            --runtime=go122 \
            --region=us-east1 \
            --source=./cmd/gcs-utils/cloud-functions/encrypt-object/ \
            --entry-point=EncryptObject \
            --trigger-topic=MetadataUpdate \
            --memory=512MB \
            --cpu=0.5 \
            --project="$PROJECT_ID"
        print_success "Cloud function deployed"
    else
        print_warning "Cloud function source not found at ./cmd/gcs-utils/cloud-functions/encrypt-object/"
        print_warning "Skipping cloud function deployment"
    fi
}

# Main execution
main() {
    echo "=================================="
    echo "  Akeso Daemon Setup Script"
    echo "=================================="
    echo
    
    print_status "Starting setup for project: $PROJECT_ID"
    echo
    
    check_prerequisites
    setup_directories
    setup_config
    enable_gcloud_services
    create_pubsub_topics
    setup_cloud_kms
    setup_iam_permissions
    build_application
    deploy_cloud_function
    
    echo
    print_success "Setup completed successfully!"
    echo
    print_status "Next steps:"
    echo "1. Review and update config/config.yaml with your specific values"
    echo "2. Ensure setupRequired is set to 'true' in config.yaml for initialization"
    echo "3. Generate or place the required key files in the keys/ directory"
    echo "4. Run the daemon with: ./akesod"
    echo "5. Set up GCS bucket notification using gcs-utils if needed"
    echo
    print_status "For GCS notification setup, use:"
    echo "./gcs-utils -notification-config -topic-id MetadataUpdate -project-id $PROJECT_ID -event-type OBJECT_METADATA_UPDATE -custom-attributes=new_dek=<YOUR_KEY> gs://<YOUR_BUCKET>"
}

# Run main function
main "$@"