# Example usages of `cloud-cp`


## Testing for CMEK
```bash
export bucket="<BUCKET>"
export project_id="<PROJECT_ID>"

# Upload
./cloud-cp -strategy cmek -cmekKey "projects/$project_id/locations/us-east1/keyRings/akeso_dev/cryptoKeys/key1" data/moby.txt gs://$bucket/moby-enc.txt

# Download
./cloud-cp -strategy cmek -cmekKey "projects/$project_id/locations/us-east1/keyRings/akeso_dev/cryptoKeys/key1/cryptoKeyVersions/1" gs://$bucket/moby-enc.txt moby-dec.txt

# Update
./cloud-cp -strategy cmek -cmekKey "projects/$project_id/locations/us-east1/keyRings/akeso_dev/cryptoKeys/key1/cryptoKeyVersions/1" -cmekUpdateKey "projects/$project_id/locations/us-east1/keyRings/akeso_dev/cryptoKeys/key3" gs://np-cmek/moby-updated.txt
```