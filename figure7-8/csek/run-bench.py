#!/usr/bin/python

import subprocess as sp
import datetime
import os
import sys
import random

STRATEGY="csek"
REPS=50

def make_folder(path):
    try:
        os.mkdir(path)
        print(f"Folder '{path}' created successfully.")
    except FileExistsError:
        print(f"Folder '{path}' already exists.")
    except Exception as e:
        print(f"An error occurred: {e}")
        
def count_lines(file_path):
    lines = 0
    with open(file_path, "r") as file:
        lines += sum(1 for _ in file)
    return lines

def create_file(file_path):
    f = open(file_path, "w")
    f.close()
    print(f"file create success:{file_path}")
    

# folder_name = datetime.datetime.now().strftime("%Y-%m-%d-%H")
folder_name = "artifact-data"
folder_path = os.path.join(".", folder_name)
make_folder(folder_path)

assert len(sys.argv) >= 2, "Number of runs required"

iter = 1
runs = int(sys.argv[1])

if len(sys.argv) >= 3:
    REPS = int(sys.argv[2])

sizes = [10240, 102400, 1048576, 10485760, 104857600]

while iter <= runs:
    
    run_path = f"{folder_path}/run-{iter}"
    make_folder(run_path)
    
    read_logs = f"{run_path}/read-logs.txt"
    with open(read_logs, 'w') as f:
        for size in sizes:
            
            data_path = f"{run_path}/{STRATEGY}-read-{size}.dat"
            create_file(data_path)
            f.write("---\n")
            f.write(f"At file: {data_path}\n")
            
            i = 1
            while i <= REPS:
                try:
                    result = sp.run(["go", "run", "benchmarks/read_full_file/main.go", "--dir", "mnt", "--file_size", f"{size}", "--duration", "0s", "--mount_cmd", "smh/mount.sh", "--raw_out", data_path], capture_output=True, text=True, check=True)
                    
                    f.write("stdout:\n")
                    f.write(result.stdout + "\n")
                    print(result.stdout)
                    
                    i += 1
                except sp.CalledProcessError as e:
                    f.write("stderr:\n")
                    f.write(e.stderr + "\n")
                    print(e.stderr)
             
    write_logs = f"{run_path}/write-logs.txt"   
    with open(write_logs, 'w') as f:
        for size in sizes:
            
            data_path = f"{run_path}/{STRATEGY}-write-{size}.dat"
            create_file(data_path)
            f.write("---\n")
            f.write(f"At file: {data_path}\n")
            
            i = 1
            while i <= REPS:
                try:
                    result = sp.run(["go", "run", "benchmarks/write_to_gcs/main.go", "--dir", "mnt", "--file_size", f"{size}", "--mount_cmd", "smh/mount.sh", "--raw_out", data_path], capture_output=True, text=True, check=True)

                    f.write("stdout:\n")
                    f.write(result.stdout + "\n")
                    print(result.stdout)
                    
                    i += 1
                except sp.CalledProcessError as e:
                    f.write("stderr:\n")
                    f.write(e.stderr + "\n")
                    print(e.stderr)
    
    iter += 1