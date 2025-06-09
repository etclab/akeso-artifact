
read_file = "nested/gcsfuse-nested/artifact-data/run-1/nested-read-10485760.dat"
write_file = "nested/gcsfuse-nested/artifact-data/run-1/nested-write-10485760.dat"

# 10 MB
size = 10485760

reads = []
writes = []
              
reads += [size/float(x) for x in open(read_file, 'r').read(-1).strip().split("\n")]
writes += [size/float(x) for x in open(write_file, 'r').read(-1).strip().split("\n")]

reads.sort()
writes.sort()

with open("plots/seqread.dat", 'w') as file:
    for element in reads:
        file.write(str(element) + '\n')

with open("plots/seqwrite.dat", 'w') as file:
    for element in writes:
        file.write(str(element) + '\n')