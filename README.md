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
