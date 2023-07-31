# disconnected-registry
This is a tool to quickly create a mirror registry for disconnected Openshift cluster for quick reproducers.

# Dependencies
- Terraform
- AWS credentials
- Pull-Secret
- Optional: Golang installed so to be able to run or compile the program.

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
The pull-secret path need to be provided with the --pull-secret flag upon running the installation of the registry. 
You can download your pull-secret from console.redhat.com

- Golang install
Please see how to download and setup Golang to your machine in [Golang Documentation](https://go.dev/doc/install)

# How to use

1) Clone the repository.
2) Run the binary directly *./terraform-registry-wrapper* or use golang to build "go build terraform-registry-wrapper.go" to create a binary executable or "go run terraform-registry-wrapper.go" to run the program directly.
   
Required flags for launching an installation:
- **--install** # Instructs the tool to that we are launching an installation.
- **--pull-secret** # Here we need to set the path to the pull-secret
- **--public-key** # With this flag we provide the path to the public-key we need to inject in the mirror-registry.

Optional flags for launching an installation:
- **--private** # This boolean is false by default. If set to true then the mirror-registry will be created with a private URL (Its VPC DNS hostname) instead of public (by default)
                # The private flag is not useful except in case one wants to bootstrap a cluster in the same VPC. I am planning to use this flag in future features.
Examples:

- **terraform-registry-wrapper** --install --pull-secret=/my/pull-secret/absolute/path --public-key=/my/public-key/absolute/path # Install a mirror registry with a public URL so it can be used from connected clusters.
- **terraform-registry-wrapper** --install --private --pull-secret=/my/pull-secret/absolute/path --public-key=/my/public-key/absolute/path # Install a mirror registry with a private URL. To be used in case one wants mirror-registry URL to be private for whatever reason.
- **terraform-registry-wrapper** --destroy # Destroy the mirror registry.

**Note:** If one creates any other resources on the created VPC manually post the deployment these need to be deleted prior running the --destroy command, terraform knows only the components that are created during the installation of the mirror-registry. Not managed objects of terraform are can cause issue when destroying the VPC.

# Additional information
The tool creates the mirror registry using the [mirror-registry](https://docs.openshift.com/container-platform/4.12/installing/disconnected_install/installing-mirroring-creating-registry.html) script.
Although it makes it much easier as it automatically makes some other actions like the below:
- Creates an EC2 instance to host the registry on AWS. (VPC,Security,Groups, Gateway, Routing etc..)
