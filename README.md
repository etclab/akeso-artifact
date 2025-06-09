### Figure 2: Key update operation using ART vs pairwise Double Ratchet key transport
- Requirements: `bash`, `python3` and `gnuplot`
- Folder: `figure2`
- Run benchmark: `./fig2.sh bench`
- Plot results: `./fig2.sh plot`
- Update submodules: `git submodule update --init --remote --recursive`

### Figure 7: Latency to read/write an entire object using encrypted cloud storage
- Requirements: 
    - publicly accessible cloud storage bucket
    - or a restricted service account that users can use to access the buckets
    - for now the service account only has read/write access to a few buckets
- Folder: `figure7`
- Run benchmark: `./fig7.sh bench`
- Plot results: `./fig7.sh plot`

### Figure 9: Time to re-encrypt a bucket of varying sizes and Figure 10: Time to re-encrypt a 1G bucket, varying object sizes in the bucket
- Requirements:
  - VM to host the `akesod` orchestrator; this can be confidential cloud VM or a trusted local server VM
- Folder: `figure9-10`
- Setup benchmark:`./fig9-10.sh setup`
- Run benchmark:`./fig9-10.sh bench`
- Plot results:`./fig9-10.sh plot`