# Disconnected-registry
This is a tool to quickly create a mirror registry and a disconnected Openshift cluster for quick reproducers regarding OLM, image management and disconnected type of reproducers.

# Description
The tool creates the mirror registry using the [mirror-registry](https://docs.openshift.com/container-platform/4.12/installing/disconnected_install/installing-mirroring-creating-registry.html) script.
Although it makes it much easier as it automatically makes some other actions like the below:
- Creates an EC2 instance to host the registry on AWS. (VPC,Security,Groups, Gateway, Routing, Certificates, oc-mirror and many more..)
- If the user creates a cluster it will download the appropriate binaries and mirror all the images for the release you have specified with the **--cluster-version** flag, then will create all required manifests, install-config.yaml and will modify some yaml files inside the manifests directory.
It does that using the provided pull-secret, public-key and some data from the terraform provissioned infrastructure the cluster is depending on.
Then the openshift-installer will be started to create the cluster.

These tasks can take a desent amount of time to setup and when the need is frequent you get the point how this project started.

# What you need to start
- Terraform
- AWS credentials
- Pull-Secret
- SSH key-pair

# How to setup
- Terraform:

Can be installed using the package manager of each distribution (yum, apt etc..)

- AWS credentials (Access Key):

Need to have AWS Credentials in ~/.aws/credentials file to be used from terraform when creating or destroying the infrastructure components.
This can be set using AWS CLI command "aws configure" or manually add the credentials under ~/.aws/credentials using the below format:
~~~
[default]
aws_access_key_id = <key-id>
aws_secret_access_key = <key-value>
~~~

- Pull-Secret and SSH key-pair:

The pull-secret path need to be provided upon running the installation of the registry the first time or by using the --init flag.
The SSH public key you will provide will be used to login to the registry node after installation.
An interactive shell with ask for the paths and will save them under the project directory in a file named "initData.json".
The --init flag is more usefull to overwrite any credentials in case the pull-secret or the key-pair are lost for example but can be used for any relevant reason.

In case you are not aware you can download your pull-secret from console.redhat.com

# How to use

1) Clone the repository.
2) Use the executable file "disconnected-wrapper" with the below flags.
   
Required flags for launching an installation of the Mirror-Registry:
- **--install** # Instructs the tool that we are launching an installation.
- **--region** # Here we set the region we like to install the Mirror-Registry. All regions available in the AWS shared account are supported. (examples: eu-west-1, eu-west-2 etc..)

Required flags for launching an installation of the Mirror-Registry AND a fully disconnected OCP cluster.
- **--install** # Instructs the tool that we are launching an installation.
- **--region** # Here we set the region we like to install the Mirror-Registry. All regions available in the AWS shared account are supported. (examples: eu-west-1,
- **--cluster** # With this flag we instruct the program that we will need to install both the registry and a disconnected OCP cluster. 
This flag is required to be provided with the below flag **--cluster-version**. Wont work without. A flag policy is also added so if you did something wrong you will get a relevant message that indicate the problem. If you find any scenario that i missed to cover please inform me to fix it.
- **--cluster-version** # With this flag we specify the exact version of the cluster we like to install (examples: 4.12.13, 4.13.11 etc..)

Credentials Initialization flag:
- **--init** # This flag i used to overwrite any credentials in case the pull-secret or the key-pair are lost or need to be changed.
It saves them under the project directory in a file named "initData.json"

Help flag:
- **--help** # It prints all flags and their descriptions.

# Additional information for the usage:

- There is a bash script for setting up the mirror-registry and the cluster (IF requested) that will be run after the creation of the registry host inside it as a terraform "user-data" script. This means that the mirror registry EC2 instance will need some time after creation to get initialized ~ 5 minutes and another ~30 minutes if a cluster is requested to finish installation.
- When the installation starts it will use terraform to create what it was told to by the flags and when terraform finishes will output the command to use to connect to the mirror-registry. For example:
~~~
ec2_instance_public_dns = "To connect to the registry run ssh -i <your-private-key> ec2-user@<public-EC2-DNS>.<region>.compute.amazonaws.com"
wait_for_initialization = "The registry requires ~ 5 minutes to initialize. It will be ready when you see the READY file under /home/ec2-user/"
~~~
So the user can login in the registry and check the progress of the deployment.
In case of only mirror-registry deployment after 5 minutes there will be a file called READY under the home folder of the ec2-user as mentioned in the output.
In either case IF the installation takes too long the user can check the script execution progress or check for any errors for troubleshooting purposes by running the below command:
~~~
$ tail -f /var/log/cloud-init-output.log
~~~

When the user logs in the registry there are 3 directories:

- mirroring-workspace # Contains a sample **imageset-config.yaml** file and oc-mirror binary
- registry-stuff # Its the registry folder as you can imagine from the name. Don't touch this directory except if you know what you are doing.
- cluster # This is the installation directory of the cluster.

# Examples:

- **disconnected-wrapper** **--init** # An interactive shell will ask you for the path of your pull-secret and your public-key. **Use absolute paths**
- **disconnected-wrapper** **--install** **--region** **eu-west-1** # Installing a Mirror-Registy in eu-west-1
- **disconnected-wrapper** **--install** **--region** **eu-west-1** **--cluster** **--cluster-version 4.12.13** # Installs a Mirror-Registry and a disconnected cluster in region eu-west-1
- **disconnected-wrapper** **--destroy** # Destroy the mirror registry.**This does not destroy the cluster IF created. User should first destroy the cluster** 
To destroy the cluster run the below command in the installation directory that is under /home/ec2-user/cluster in the created Registry instance.

**Important:** The cluster is not managed by this tool upon creation only the Mirror-Registry is. It is the user responsibility to first destroy the cluster using the openshift-installer prior running **--destroy**.
This is because if the instance get destroyed prior the cluster the installation directory will get lost so all the resources of the cluster will remain running on AWS and a manual cleanup will be needed that is not the best experience if you ask me. 
For this reason i added an interactive question to ask the user every time **--destroy** flag is used before it destroy the Mirror-Registry. 
Only with "yes" will destroy.

**Note:** If one creates any other resources on the created VPC manually post the deployment these need to be deleted prior running the --destroy command, terraform knows only the components that are created during the installation of the mirror-registry. Not managed objects of terraform can cause issues when destroying the VPC.

# Usefull Information

- The cluster directory is under /home/ec2-user/cluster/
- The cluster SSH key is under /home/ec2-user/.ssh/cluster_key
- Mirror registry has SSH access to all nodes
- The kubeconfig of the cluster is under /home/ec2-user/cluster/auth/kubeconfig
- One can check the installer progress by running tail -f /home/ec2-user/cluster/.openshift-install.log