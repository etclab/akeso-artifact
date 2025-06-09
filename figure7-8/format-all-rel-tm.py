#!/usr/bin/python

import os
import statistics
import numpy as np
import pprint

# find the average bytes/sec for each object size
# output the result to dat file in following format
# ----------
# Size      rs_m    ws_m    rs_std  ws_std  rs_50  ws_50 rs_90  ws_90 rs_99  ws_99
# 10K       x       y       z       b       c      d     e      f     g      h
# 100K      x       y       z       b       c      d     e      f     g      h
# 1M        x       y       z       b       c      d     e      f     g      h
# 10M       x       y       z       b       c      d     e      f     g      h
# 100M      x       y       z       b       c      d     e      f     g      h

# object size in bytes
sizes = [10 * 1024, 100 * 1024, 1024 * 1024, 10 * 1024 * 1024, 100 * 1024 * 1024]
size_name = { 10240: '10K', 102400: '100K', 1048576: '1M',
             10485760: '10M', 104857600: '100M'}

# nested is the akeso strategy
strategies = ["cmek", "cmek-hsm", "csek", "keywrap", "strawman", "nested"]

script_dir = os.path.dirname(os.path.abspath(__file__))

cmek_data = {}

for strat in strategies:
    for run in range(1,2):        
        # tm = trimmed mean
        file = f"{script_dir}/plots/{strat}-all-tm.dat"
        
        f = open(file, 'w')
        f.write(f"# {strat} run-{run} (seconds)\n")
        f.write(f"{'# size':<8} {'rs_m':<25} {'ws_m':<25} {'rs_std':<25} {'ws_std':<25} {'rs_50':<25} {'ws_50':<25} {'rs_90':<25} {'ws_90':<25} {'rs_99':<25} {'ws_99':<25}\n")
        
        for size in sizes:    
            reads = []
            writes = []
            
            read_file = f"{script_dir}/{strat}/gcsfuse-{strat}/artifact-data/run-{run}/{strat}-read-{size}.dat"
            write_file = f"{script_dir}/{strat}/gcsfuse-{strat}/artifact-data/run-{run}/{strat}-write-{size}.dat"
            
            reads += [size/float(x) for x in open(read_file, 'r').read(-1).strip().split("\n")]
            writes += [size/float(x) for x in open(write_file, 'r').read(-1).strip().split("\n")]
                    
            reads.sort()
            writes.sort()
            
            # discard the first and last 1 values
            reads = reads[1:-1]
            writes = writes[1:-1]
            
            rs_m = statistics.mean(reads)
            ws_m = statistics.mean(writes)

            rs_std = statistics.stdev(reads)
            ws_std = statistics.stdev(writes)
            
            rs_50 = np.percentile(reads, 50).item()
            ws_50 = np.percentile(writes, 50).item()
            rs_90 = np.percentile(reads, 90).item()
            ws_90 = np.percentile(writes, 90).item()
            rs_99 = np.percentile(reads, 99).item()
            ws_99 = np.percentile(writes, 99).item()
            
            if strat == "cmek":
                if f"run-{run}" not in cmek_data:
                    cmek_data[f"run-{run}"] = {}
                
                if f"{size}" not in cmek_data[f"run-{run}"]:
                    cmek_data[f"run-{run}"][f"{size}"] = {}
                    
                cmek_run_size = cmek_data[f"run-{run}"][f"{size}"]
                cmek_run_size["rs_m"] = rs_m
                cmek_run_size["ws_m"] = ws_m
                cmek_run_size["rs_std"] = rs_std
                cmek_run_size["ws_std"] = ws_std
                cmek_run_size["rs_50"] = rs_50
                cmek_run_size["ws_50"] = ws_50
                cmek_run_size["rs_90"] = rs_90
                cmek_run_size["ws_90"] = ws_90
                cmek_run_size["rs_99"] = rs_99
                cmek_run_size["ws_99"] = ws_99
                
        
            f.write(f"{size_name[size]:<8} {rs_m:<25} {ws_m:<25} {rs_std:<25} {ws_std:<25} {rs_50:<25} {ws_50:<25} {rs_90:<25} {ws_90:<25} {rs_99:<25} {ws_99:<25}\n")

        f.close() 
        
for strat in strategies:
    for run in range(1,2):
        rel_file = f"{script_dir}/plots/{strat}-all-tm-rel.dat" # relative to CMEK
        
        frel = open(rel_file, 'w')
        frel.write(f"# {strat} run-{run} relative to CMEK \n")
        frel.write(f"{'# size':<8} {'rs_m':<25} {'ws_m':<25} {'rs_std':<25} {'ws_std':<25} {'rs_50':<25} {'ws_50':<25} {'rs_90':<25} {'ws_90':<25} {'rs_99':<25} {'ws_99':<25}\n")
        
        for size in sizes:    
            reads = []
            writes = []
            
            read_file = f"{script_dir}/{strat}/gcsfuse-{strat}/artifact-data/run-{run}/{strat}-read-{size}.dat"
            write_file = f"{script_dir}/{strat}/gcsfuse-{strat}/artifact-data/run-{run}/{strat}-write-{size}.dat"
            
            reads += [size/float(x) for x in open(read_file, 'r').read(-1).strip().split("\n")]
            writes += [size/float(x) for x in open(write_file, 'r').read(-1).strip().split("\n")]
                    
            reads.sort()
            writes.sort()
            
            # discard the first and last 1 values
            reads = reads[1:-1]
            writes = writes[1:-1]
            
            cmek_datum = cmek_data[f"run-{run}"][f"{size}"]
            
            rs_m = statistics.mean(reads) / cmek_datum["rs_m"]
            ws_m = statistics.mean(writes) / cmek_datum["ws_m"]

            rs_std = statistics.stdev(reads) / cmek_datum["rs_std"]
            ws_std = statistics.stdev(writes) / cmek_datum["ws_std"]
            
            rs_50 = np.percentile(reads, 50).item() / cmek_datum["rs_50"]
            ws_50 = np.percentile(writes, 50).item() / cmek_datum["ws_50"]
            rs_90 = np.percentile(reads, 90).item() / cmek_datum["rs_90"]
            ws_90 = np.percentile(writes, 90).item() / cmek_datum["ws_90"]
            rs_99 = np.percentile(reads, 99).item() / cmek_datum["rs_99"]
            ws_99 = np.percentile(writes, 99).item() / cmek_datum["ws_99"]
            
            frel.write(f"{size_name[size]:<8} {rs_m:<25} {ws_m:<25} {rs_std:<25} {ws_std:<25} {rs_50:<25} {ws_50:<25} {rs_90:<25} {ws_90:<25} {rs_99:<25} {ws_99:<25}\n")
            
        frel.close()