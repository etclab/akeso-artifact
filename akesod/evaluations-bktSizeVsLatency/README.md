# Evaluations of Time to re-encrypt bucket sizes of varying sizes with 2MB object sizes

## Steps:

1. Run the evaluations for n times as: 

```bash
bash automate.sh 10 # running 10 times
```

2. Extract the data from multiple runs

```bash
python3 process-data.py
```

Note: `process-data.py` requires `numpy`, make sure the Python environment has it installed.