import re

def process(in_file, out_file, pattern):
    with open(in_file, "r") as f:
        data = []
        for line in f:
            match = pattern.search(line)
            if match:
                data.append(line.strip())
        
        out = open(out_file, 'w')
        out.write(f"{'# group size':<15} {'ns/op (op = key update)':<25}\n")
        
        for line in data:
            cols = line.split()
            op_time = cols[2]
            op_raw = op_time.split()[0]
            
            group_size = cols[0].split('-')[1]
            
            out.write(f"{group_size:<15} {op_raw:<25}\n")
            
        out.close()

outputs = ["art.dat", "drat.dat"]
inputs = ["art.bench", "drat.bench"]

# for art.bench
in_file = "art.bench"
out_file = "art.dat"
pattern = re.compile(r"BenchmarkUpdate/Update-\d+-\d+")
process(in_file, out_file, pattern)

# for drat.bench
in_file = "drat.bench"
out_file = "drat.dat"
pattern = re.compile(r"BenchmarkKeyRotation/Group-\d+-\d+")
process(in_file, out_file, pattern)


