# disconnected-registry
This is a tool to quickly create a mirror registry and a disconnected Openshift cluster for quick reproducers.

# Additional information
The tool creates the mirror registry using the [mirror-registry](https://docs.openshift.com/container-platform/4.12/installing/disconnected_install/installing-mirroring-creating-registry.html) script.
Although it makes it much easier as it automatically makes some other actions like the below:
- Creates an EC2 instance to host the registry on AWS. (VPC,Security,Groups, Gateway, Routing, Certificates, oc-mirror and many more..)

# Dependencies
- Terraform
- AWS credentials
- Pull-Secrets
- SSH key-pair
- Optional: Golang installed so to be able to run or compile the program. One can get the binary directly from the repo.

# How to setup
- Terraform:
Can be installed using the package manager of each distribution (yum, apt etc..)

- AWS credentials (Access Key)
Need to have AWS Credentials in ~/.aws/credentials file to be used from terraform when creating or destroying the infrastructure components.
This can be set using AWS CLI command "aws configure" or manually add the credentials under ~/.aws/credentials using the below format:
~~~
[default]
aws_access_key_id = <key-id>
aws_secret_access_key = <key-value>
~~~

- Pull-Secret and SSH key-pair
The pull-secret path need to be provided upon running the installation of the registry the first time or by using the --init flag.
The SSH public key you will provide will be used to login to the registry node after installation.
An interactive shell with ask for the paths and will save them under the project directory in a file named "initData.json"
The --init flag is more usefull to overwrite any credentials in case the pull-secret or the key-pair are lost for example but can be used for any relevant reason.

In case you are not aware you can download your pull-secret from console.redhat.com

- Golang install
Please see how to download and setup Golang to your machine in [Golang Documentation](https://go.dev/doc/install)
It is not mandatory cause you can use directly the compiled binary of the program.

# How to use

1) Clone the repository.
2) Use golang to build "go build terraform-registry-wrapper.go" to create a binary executable OR "go run terraform-registry-wrapper.go" to run the program directly OR 
use the binary directly
   
Required flags for launching an installation of the Mirror-Registry:
- **--install** # Instructs the tool that we are launching an installation.
- **--region** # Here we set the region we like to install the Mirror-Registry. All regions available in the AWS shared account are supported. (examples: eu-west-1, eu-west-2 etc..)

Required flags for launching an installation of the Mirror-Registry AND a fully disconnected OCP cluster.
- **--cluster** # With this flag we instruct the program that we will need to install both the registry and a disconnected OCP cluster. 
This flag is required to be provided with the below flag **--cluster-version**. Wont work without.
- **--cluster-version** # With this flag we specify the exact version of the cluster we like to install (examples: 4.12.13, 4.13.11 etc..)

Credentials Initialization flag:
- **--init** # This flag i used to overwrite any credentials in case the pull-secret or the key-pair are lost or need to be changed.
It saves them under the project directory in a file named "initData.json"

Small Summary of what the tool does:

The user depending the flags used will trigger a series of automated tasks like, Mirror-Registry will get created, it will download the appropriate binaries and mirror all the images for the release you have specified with the **--cluster-version** flag, then will create all required manifests (install-config.yaml and will modify some yaml files inside the manifests dir)
It does that using the provided pull-secret, public-key and some data from the terraform provissioned infrastructure the cluster is depending on.
Then the openshift-installer will be started to create the cluster.

Examples:

- **terraform-registry-wrapper** **--init** # An interactive shell will ask you for the path of your pull-secret and your public-key.
- **terraform-registry-wrapper** **--install** **--region** **eu-west-1** # Installing a Mirror-Registy in eu-west-1
- **terraform-registry-wrapper** **--install** **--region** **--cluster** **--cluster-version 4.12.13** # Installs a Mirror-Registry and a disconnected cluster.
- **terraform-registry-wrapper** **--destroy** # Destroy the mirror registry.

**Important:** The cluster is not managed by this tool upon creation only the Mirror-Registry is. It is the user responsibility to first destroy the cluster using the openshift-installer prior running **--destroy**.
This is because if the instance get destroyed prior the cluster the installation directory will get lost so all the resources of the cluster will remain running on AWS and a manual cleanup will be needed that is not the best experience if you ask me. 
For this reason i added an interactive question to ask the user every time **--destroy** flag is used before it destroy the Mirror-Registry. 
Only with "yes" will destroy.

**Note:** If one creates any other resources on the created VPC manually post the deployment these need to be deleted prior running the --destroy command, terraform knows only the components that are created during the installation of the mirror-registry. Not managed objects of terraform can cause issues when destroying the VPC.
