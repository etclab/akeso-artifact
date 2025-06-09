#!/bin/bash

# this script sets up a service account with required iam policies for accessing
# gcloud buckets, subscriptions and topics

# all of these commands need to be run from an admin account that has the 
# necessary permissions to add iam policies to the service account eg. roles/pubsub.admin

# create service account
gcloud iam service-accounts create ae-pets25-alice \
  --description="Artifact Evaluator Alice" \
  --display-name="AE Alice PETS25"

# the service account needs access to gs://atp-csek
gcloud storage buckets add-iam-policy-binding gs://atp-csek \
    --member="serviceAccount:ae-pets25-alice@ornate-flame-397517.iam.gserviceaccount.com" \
    --role="roles/storage.objectUser"

# the service account needs access to gs://atp-cmek
gcloud storage buckets add-iam-policy-binding gs://atp-cmek \
    --member="serviceAccount:ae-pets25-alice@ornate-flame-397517.iam.gserviceaccount.com" \
    --role="roles/storage.objectUser"

# the service account needs access to gs://atp-cmek-hsm
gcloud storage buckets add-iam-policy-binding gs://atp-cmek-hsm \
    --member="serviceAccount:ae-pets25-alice@ornate-flame-397517.iam.gserviceaccount.com" \
    --role="roles/storage.objectUser"

# the service account needs access to gs://atp-keywrap
gcloud storage buckets add-iam-policy-binding gs://atp-keywrap \
    --member="serviceAccount:ae-pets25-alice@ornate-flame-397517.iam.gserviceaccount.com" \
    --role="roles/storage.objectUser"

# the service account needs access to gs://atp-nested
gcloud storage buckets add-iam-policy-binding gs://atp-nested \
    --member="serviceAccount:ae-pets25-alice@ornate-flame-397517.iam.gserviceaccount.com" \
    --role="roles/storage.objectUser"

# the service account needs access to gs://atp-strawman
gcloud storage buckets add-iam-policy-binding gs://atp-strawman \
    --member="serviceAccount:ae-pets25-alice@ornate-flame-397517.iam.gserviceaccount.com" \
    --role="roles/storage.objectUser"

# grant permissions for the topics
gcloud pubsub topics add-iam-policy-binding atp-group-setup \
    --member="serviceAccount:ae-pets25-alice@ornate-flame-397517.iam.gserviceaccount.com" \
    --role="roles/pubsub.publisher"

gcloud pubsub topics add-iam-policy-binding atp-key-update \
    --member="serviceAccount:ae-pets25-alice@ornate-flame-397517.iam.gserviceaccount.com" \
    --role="roles/pubsub.publisher"

# grant permissions for the subscriptions
gcloud pubsub subscriptions add-iam-policy-binding atp-group-setup-bob \
    --member="serviceAccount:ae-pets25-alice@ornate-flame-397517.iam.gserviceaccount.com" \
    --role="roles/pubsub.subscriber"

gcloud pubsub subscriptions add-iam-policy-binding atp-key-update-bob \
    --member="serviceAccount:ae-pets25-alice@ornate-flame-397517.iam.gserviceaccount.com" \
    --role="roles/pubsub.subscriber"

# activate the service account in gcloud
gcloud auth activate-service-account --key-file=serviceAccount-ae-pets25-alice.json

# additionally set the key file for service account 
export GOOGLE_APPLICATION_CREDENTIALS=$HOME/downloads/artifact-akeso/serviceAccount-ae-pets25-alice.json