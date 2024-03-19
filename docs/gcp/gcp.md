### Table of Contents ###

### Current functional egress flags for GCP are subnet-id, instance-type, region, cloud-tags, debug, timeout ###
### TODO Add features - egress flags - image-id, kms-key-id for GCP ###

- [Setup](#setup)
  - [GCP Environment](#gcp-environment)
- [Available tools](#available-tools)
  - [1. Egress Verification](#1-egress-verification)
    - [1.1 Usage](#11-usage)
      - [1.1.1 CLI Executable](#111-cli-executable)

## Setup ##
### GCP Environment ###
Set up your environment to use the correct VPC name, project ID, credentials of the GCP account for the target cluster.
- Make sure to have a Service Account with the permissions required within your GCP account (in that project). This can be done in the following ways:
  -  Follow the steps to use a script as prescibed in [this document.](https://github.com/openshift/ops-sop/blob/master/gcp/create-ccs-project.md) or create a service account manually [this] (https://cloud.google.com/iam/docs/creating-managing-service-accounts#iam-service-accounts-create-gcloud)
  - Export these GCP environment variables:
     ```shell
     export GCP_VPC_NAME=<YOUR_GCP_VPC_NAME)>
     export GCP_PROJECT_ID=<YOUR_GCP_PROJECT_ID>
     ```
    Export any other GCP environment vars:
      ```shell
      export GCP_REGION=<VPC_GCP_REGION>
      export GOOGLE_APPLICATION_CREDENTIALS=<PATH_TO_CREDENTIALS_JSON_FILE>
      ````
  
### IAM permissions ###
Ensure that the GCP credentials being used have the following permissions:
![image](https://user-images.githubusercontent.com/77566186/179435749-0fc92102-21e5-43e8-a401-32cabeb19f56.png)
 
## Available Tools ##

### 1. Egress Verification ###
#### 1.1 Usage ####
The processes below describe different ways of using egress verifier on a single subnet. 
In order to verify entire VPC, 
repeat the verification process for each subnet ID.

##### 1.1.1 CLI Executable #####
   1. Ensure correct [environment setup](#setup).

   2. Clone the source:
      ```shell
      git clone https://github.com/openshift/osd-network-verifier.git
      ``` 
   3. Build the cli:
      ```shell
      make build
      ```
      This generates `osd-network-verifier` executable in project root directory. 

   4. Obtain params:
      1. subnet_id: Obtain the subnet id to be verified. 

   5. Execute:

       ```shell        
      # GCP
      ./osd-network-verifier egress --platform gcp-classic --subnet-id $SUBNET_ID 
      
        Additional optional flags for overriding defaults (image-id, kms-key will be added in the future):
      ```shell
      --cloud-tags stringToString   (optional) comma-seperated list of tags to assign to cloud resources e.g. --cloud-tags key1=value1,key2=value2 (default [osd-network-verifier=owned,red-hat-managed=true,Name=osd-network-verifier])
      --debug                       (optional) if true, enable additional debug-level logging
      -- TODO image-id string             (optional) cloud image for the compute instance
      --instance-type string        (optional) compute instance type (default "e2-standard-2")
      -- TODO kms-key-id string           (optional) ID of KMS key used to encrypt root volumes of compute instances. Defaults to cloud account default key
      --region string               (optional) compute instance region. If absent, environment var GCP_REGION will be used, if set (default "us-east1")
      
      --subnet-id string            source subnet ID
      --timeout duration            (optional) timeout for individual egress verification requests (default 2s). If timeout is less than 2s, it would likely cause false negatives test results.
         ```
   
       Get cli help:
    
        ```shell
        ./osd-network-verifier egress --help
        ```
