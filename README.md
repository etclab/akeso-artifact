## Akeso: Bringing Post-Compromise Security to Cloud Storage

Here's the list of components that make up Akeso
- **gcsfuse** - [Link](https://github.com/etclab/gcsfuse)
  - Our fork is based on Google's open source Cloud Storage FUSE 
  ([`gcsfuse@3525748`](https://github.com/etclab/gcsfuse/commit/3525748)) 
  - Implements Akeso’s strategies on top of `gcsfuse` to transparently encrypt 
  and decrypt objects during read and write operations.
- **art** - [Link](https://github.com/etclab/art)
  - Implements the Asynchronous Ratcheting Tree (ART) data structure and 
  associated protocols
- **nestedaes** - [Link](https://github.com/etclab/nestedaes)
  - Implements the updatable re-encryption using nested AES
- **akesod** - [Link](./akesod/)
  - Manages group membership, and generates the re-encryption tokens
- **akeso-evals** - [Link](https://github.com/etclab/akeso-evals)
  - Includes various utility scripts for parsing data, plotting figures, 
  running pre-evaluation, etc. 

- **Build Environment**
  - We developed and tested Akeso and its components on Ubuntu 24.04 LTS, but it should work correctly on other systems as long as the required packages are installed. 
  - All components listed above can be run with the following dependencies: `go, gnuplot, fuse3, and python3`. 
  - The required packages can be installed using the command below: 
    ```bash
    ./common/install-dependencies.sh && ./common/install-go.sh
    ```

### (Minor) Differences from paper
- TODO: mine differences from the paper


## Reproducing Experiments
- The required packages can be installed using the command below (note: please skip `./common/install-go.sh` if you already have `Go` installed - as it'll replace the `Go` on your path, and `./common/install-gcloud.sh` if you already have gcloud cli installed):
  ```bash
  ./common/install-dependencies.sh && ./common/install-go.sh && ./common/install-gcloud.sh && source ~/.bashrc
  ``` 

### Figure 2: Key update operation using ART vs pairwise Double Ratchet key transport
- Requirements: `bash`, `python3` and `gnuplot`
- Folder: `figure2`
- Run benchmark: `./fig2.sh bench`
- Plot results: `./fig2.sh plot`
- Update submodules: `git submodule update --init --remote --recursive`

### Figure 7: Latency to read/write an entire object using encrypted cloud storage
- Requirements: 
    - Local packages: `bash`, `python3`, `Go`, `gnuplot`, `gcloud`
    - Cloud resources
      - Cloud Storage buckets hosted in Google cloud
      - To access the buckets, a service account key file with necessary access to buckets, keys, pub/sub topics and subscriptions is included in the HotCRP
      - Setup Service Account (SA) credentials to access the cloud resource.
        ```bash
        # adjust the service account key path accordingly
        export GOOGLE_APPLICATION_CREDENTIALS=$HOME/downloads/serviceAccount-ae-pets25-alice.json
        gcloud auth activate-service-account --key-file=$GOOGLE_APPLICATION_CREDENTIALS
        ```
- Folder: `figure7-8`
- Run benchmark: `./fig7.sh bench`
- Plot results: `./fig7.sh plot`

### Figure 8: CDF of Latencies to read and write a 10MB object with Akeso
- Figure 8 is created using data from experiment for Figure 7
- Plot results: `./fig8.sh plot`

### Figure 9: Time to re-encrypt a bucket of varying sizes and Figure 10: Time to re-encrypt a 1G bucket, varying object sizes in the bucket
- Requirements:
  - VM to host the `akesod` orchestrator; this can be confidential cloud VM or a trusted local server VM
- Folder: `figure9-10`
- Setup benchmark:`./fig9-10.sh setup`
- Run benchmark:`./fig9-10.sh bench`
- Plot results:`./fig9-10.sh plot`

