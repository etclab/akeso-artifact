# Artifact Appendix

Paper title: **Akeso: Bringing Post-Compromise Security to Cloud Storage**

Artifacts HotCRP Id: **#31** 

Requested Badge: **Available**

## Description

The artifact for Akeso is distributed into five repositories, all hosted on GitHub. Here's a short description of how each component fits into the design mentioned in the paper. 
- `art` implements the Asynchronous Ratcheting Tree (ART) data structure used for group key generation
- `nestedaes` implements the updated re-encryption using nested AES
- `akesod` runs on the cloud handling group membership as well as re-encryption token generation
- `gcsfuse` includes the six different encryption strategies used with a cloud storage bucket
- `akeso-evals` includes various scripts, data, and instructions for running the benchmarks 
- `akeso-artifact` (this repository) brings together the above components into a single repository

### Security/Privacy Issues and Ethical Concerns (All badges)

The artifact does not contain any malware samples or pose any risk to the security or privacy of the reviewer's machine. Furthermore, there are no ethical concerns associated with running the artifact.

<!-- ## Basic Requirements (Only for Functional and Reproduced badges)
Describe the minimal hardware and software requirements of your artifact and estimate the compute time and storage required to run the artifact.

### Hardware Requirements
If your artifact requires specific hardware to be executed, mention that here.
Provide instructions on how a reviewer can gain access to that hardware through remote access, buying or renting, or even emulating the hardware.
Make sure to preserve the anonymity of the reviewer at any time.

### Software Requirements
Describe the OS and software packages required to evaluate your artifact.
This description is essential if you rely on proprietary software or software that might not be easily accessible for other reasons.
Describe how the reviewer can obtain and install all third-party software, data sets, and models.

### Estimated Time and Storage Consumption
Provide an estimated value for the time the evaluation will take and the space on the disk it will consume. 
This helps reviewers to schedule the evaluation in their time plan and to see if everything is running as intended.
More specifically, a reviewer, who knows that the evaluation might take 10 hours, does not expect an error if, after 1 hour, the computer is still calculating things. -->

## Environment 
In the following, describe how to access our artifact and all related and necessary data and software components.
Afterward, describe how to set up everything and how to verify that everything is set up correctly. 

### Accessibility (All badges)
Valid hosting options are institutional and third-party digital repositories.
Do not use personal web pages.
For repositories that evolve over time (e.g., Git Repositories ), specify a specific commit-id or tag to be evaluated.
In case your repository changes during the evaluation to address the reviewer's feedback, please provide an updated link (or commit-id / tag) in a comment.

The artifact is hosted on GitHub and made publicly available at https://github.com/etclab/akeso-artifact. The artifact repository consists of code, data, and instructions for running Akeso and its components.

The `gcsfuse` client can run on any recent Linux system. However, components such as `akesod`, its communication with the client, group operations, and re-encryption rely on Google Cloud services—including Pub/Sub, Cloud Storage, Cloud Run, and Compute Engine.

We have provided the necessary cloud resources and scripts to run the individual components and experiments, wherever applicable.

<!-- ### Set up the environment (Only for Functional and Reproduced badges)
Describe how the reviewers should set up the environment for your artifact, including downloading and installing dependencies and the installation of the artifact itself.
Be as specific as possible here.
If possible, use code segments to simply the workflow, e.g.,

```bash
git clone git@my_awesome_artifact.com/repo
apt install libxxx xxx
```
Describe the expected results where it makes sense to do so.

### Testing the Environment (Only for Functional and Reproduced badges)
Describe the basic functionality tests to check if the environment is set up correctly.
These tests could be unit tests, training an ML model on very low training data, etc..
If these tests succeed, all required software should be functioning correctly.
Include the expected output for unambiguous outputs of tests.
Use code segments to simplify the workflow, e.g.,
```bash
python envtest.py
```

## Artifact Evaluation (Only for Functional and Reproduced badges)
This section includes all the steps required to evaluate your artifact's functionality and validate your paper's key results and claims.
Therefore, highlight your paper's main results and claims in the first subsection. And describe the experiments that support your claims in the subsection after that.

### Main Results and Claims
List all your paper's results and claims that are supported by your submitted artifacts.

#### Main Result 1: Name
Describe the results in 1 to 3 sentences.
Refer to the related sections in your paper and reference the experiments that support this result/claim.

#### Main Result 2: Name
...

### Experiments 
List each experiment the reviewer has to execute. Describe:
 - How to execute it in detailed steps.
 - What the expected result is.
 - How long it takes and how much space it consumes on disk. (approximately)
 - Which claim and results does it support, and how.

#### Experiment 1: Name
Provide a short explanation of the experiment and expected results.
Describe thoroughly the steps to perform the experiment and to collect and organize the results as expected from your paper.
Use code segments to support the reviewers, e.g.,
```bash
python experiment_1.py
```
#### Experiment 2: Name
...

#### Experiment 3: Name 
...

## Limitations (Only for Functional and Reproduced badges)
Describe which tables and results are included or are not reproducible with the provided artifact.
Provide an argument why this is not included/possible.

## Notes on Reusability (Only for Functional and Reproduced badges)
First, this section might not apply to your artifacts.
Use it to share information on how your artifact can be used beyond your research paper, e.g., as a general framework.
The overall goal of artifact evaluation is not only to reproduce and verify your research but also to help other researchers to re-use and improve on your artifacts.
Please describe how your artifacts can be adapted to other settings, e.g., more input dimensions, other datasets, and other behavior, through replacing individual modules and functionality or running more iterations of a specific part. -->
