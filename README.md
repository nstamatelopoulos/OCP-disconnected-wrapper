# disconnected-registry
This is a tool to quickly create a mirror registry for disconnected Openshift cluster for quick reproducers.

# Dependencies
- Terraform
- AWS credentials
- Pull-Secret
- Golang installed so to be able to run or compile the program.

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

- Pull-Secret
The pull-secret need to be added inside the config directory and to have the name pull-secret.json in beautified format.
You can download your pull-secret from console.redhat.com and to beutify it using "jq" command. The below command should be sufficient:
~~~
$ cat pull-secret.txt | jq > pull-secret.json
~~~
- Golang install
Please see how to download and setup Golang to your machine in [Golang Documentation](https://go.dev/doc/install)

# How to use

1) Clone the repository and add in the config directory the pull-secret.json file as mentioned above.
2) Run "go build terraform-registry-wrapper.go" to create a binary executable or "go run terraform-registry-wrapper.go" to run the program directly.
Options:
- **terraform-registry-wrapper** --install # Install a mirror registry with a public URL so it can be used from connected clusters.
- **terraform-registry-wrapper** --install --private # Install a mirror registry with a private URL. To be used in case one wants mirror-registry URL to be private.
- **terraform-registry-wrapper** --destroy # Destroy the mirror registry.

**Note:** Do not create any other resources on the created VPC manually post deployment as all components are managed by terraform and it can cause issue when destroying. If manually resources were created
it will be required to be manually deleted and run destroy again.

# Additional information
The tool creates the mirror registry using the [mirror-registry](https://docs.openshift.com/container-platform/4.12/installing/disconnected_install/installing-mirroring-creating-registry.html) script.
Although it makes it much easier as it automatically makes some other actions like the below:
- Creates an EC2 instance to host the registry on AWS. (VPC,Security,Groups,SSH key etc..)
- After creating the registry it automatically builds the pull-secret with the regitry credentials and store it under ~/.docker/config.json (Default location for oc-mirror tool to use it)
- Configures the SSL trust between the host and the registry container by adding the CA of the self-signed cert in the system CA store.
- Downloads and setup the oc-mirror tool to be used for mirroring.
- Creates SSH key and provides the SSH private key under /config/ directory to be used for logging in to the registry EC2. (After successfull creation the ssh command is getting printed for ease of use)
Example:
~~~
aws_instance.mirror-registry (remote-exec): SSH command to registry host ssh -i ./config/awsRegistrySSHKey ec2-user@ec2-XX-XX-XXX-XX.eu-west-1.compute.amazonaws.com
~~~
