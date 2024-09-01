package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	registryScriptTemplate = "registry-mirror-script-terraform.sh.temp"
	registryScript         = "registry-mirror-script-terraform.tpl"
	pullSecretTemplate     = "pull-secret.template"
	initFileName           = "initData.json"
	CAcert                 = "CAcert.pem"
)

// All RHEL 9 AMI images for all regions under our AWS lab account

var regions = map[string]string{
	"eu-west-1":      "ami-07d4917b6f95f5c2a",
	"eu-west-2":      "ami-07d1e0a32156d0d21",
	"eu-west-3":      "ami-0574a94188d1b84a1",
	"eu-central-1":   "ami-007c3072df8eb6584",
	"eu-south-2":     "ami-05cdcc0c8c82bd18e",
	"eu-north-1":     "ami-064983766e6ab3419",
	"us-east-1":      "ami-0583d8c7a9c35822c",
	"us-east-2":      "ami-0aa8fc2422063977a",
	"us-west-1":      "ami-0c5ebd68eb61ff68d",
	"us-west-2":      "ami-0423fca164888b941",
	"ap-south-1":     "ami-022ce6f32988af5fa",
	"ap-northeast-3": "ami-033c6909beae3b794",
	"ap-northeast-2": "ami-012e764b9ddef07c2",
	"ap-southeast-1": "ami-0b748249d064044e8",
	"ap-southeast-2": "ami-086918d8178bfe266",
	"ap-northeast-1": "ami-04d3ba818c434b384",
	"ca-central":     "ami-0775d166d9bde92c8",
	"sa-east-1":      "ami-06dec7e27b4abea7b",
}

var pullSecretPath string
var publicKeyPath string

func main() {

	// All flags that make this tool.
	region := flag.String("region", "", "Set the AWS region")
	installFlag := flag.Bool("install", false, "Install Registry")
	destroyFlag := flag.Bool("destroy", false, "Destroy Registry")
	clusterVersion := flag.String("cluster-version", "", "Set the prefered cluster version")
	initFlag := flag.Bool("init", false, "Saving pull-secret and public-key for ease of use")
	openshiftCNI := flag.Bool("sdn", false, "Use SDN CNI for the cluster instead. OVN is the default")
	helpFlag := flag.Bool("help", false, "Help")
	statusFlag := flag.Bool("status", false, "Status of the deployment")
	addClusterFlag := flag.Bool("add-cluster", false, "To deploy a cluster but keep the existing registry")
	destroyClusterFlag := flag.Bool("destroy-cluster", false, "To destroy the cluster but keep the existing registry")
	installConfigFlag := flag.Bool("custom-install-config", false, "Edit the default install-config.yaml")
	forceFlag := flag.Bool("force", false, "Force destroy the infrastructure if agent is unavailable. (Terraform destroy)")

	flag.Parse()

	// This is a function that has policies for all flags to prevent program failure if user provide them incorectly. Check flags.go package for the code.
	consolidatedFlagCheckFunction(*installFlag, *destroyFlag, *region, *clusterVersion, *initFlag, *helpFlag, *openshiftCNI, *destroyClusterFlag, *addClusterFlag, *installConfigFlag, *forceFlag)

	// Here we handle the case where the user will attempt to add a cluster when a registry host is already provisioned.
	if *addClusterFlag && len(*clusterVersion) > 0 {
		GetInfraDetails()
		agentRegistryStatus := ClientGetStatus(infraDetailsStatus.InstancePublicDNS)
		if agentRegistryStatus {
			applyTerraformConfig()
			GetInfraDetails()
			installConfig := populateInstallConfigValues(*openshiftCNI, *installConfigFlag)
			sendInstallConfigToAgent(installConfig, infraDetailsStatus.InstancePublicDNS)
			populateActionAndVersion(true, *clusterVersion)
			sendActionAndVersionToAgent(infraDetailsStatus.InstancePublicDNS)
		} else if agentStatus.ClusterStatus == "Exists" {
			fmt.Println("There is already an existing cluster installation present and cannot deploy a new one")
		} else {
			fmt.Println("Agent or Registry unhealthy")
		}
		return
	}

	// Here we hadle the case where the user will attempt to destroy a cluster ONLY. Not the registry host too.
	if *destroyClusterFlag {
		GetInfraDetails()
		agentRegistryStatus := ClientGetStatus(infraDetailsStatus.InstancePublicDNS)
		if agentRegistryStatus && agentStatus.ClusterStatus == "Exists" {
			populateActionAndVersion(false, *clusterVersion)
			sendActionAndVersionToAgent(infraDetailsStatus.InstancePublicDNS)
		} else if agentStatus.ClusterStatus == "DontExist" {
			fmt.Println("There is no cluster installation present.")
		} else {
			fmt.Println("Agent or Registry unhealthy")
		}
		return
	}

	// Here we can use the --status flag to get information on what we have provisioned and what not. Returns if a cluster is present and if QUAY registry is healhty.
	if *statusFlag {
		GetInfraDetails()
		ClientGetStatus(infraDetailsStatus.InstancePublicDNS)
	}

	// If init flag is used then start interactive prompt to get the paths
	if *initFlag {
		initialization(initFileName)
	}
	// If the help flag is used display the flag descriptions
	if *helpFlag {
		flagsHelp()
	}

	// If the install flag is used do appropriate actions for installation
	if *installFlag {

		// Check if there is already installed infrastructure before you redeploy.
		if _, err := os.Stat("./terraform.tfstate"); os.IsNotExist(err) {
			fmt.Println("No terraform.tfstate file detected. The tool is probably run for the first time")
		} else if err == nil {
			fmt.Println("The terraform.tfstate file is detected. Checking current state.")
		} else {
			fmt.Println("Error:", err)
		}

		// Delete left over templates in case it was not cleaned up properly. Normally this should not be required but adding just in case
		deleteGeneratedFiles()

		// Check if the credentials are present if not ask for them
		if _, err := os.Stat(initFileName); os.IsNotExist(err) {
			fmt.Println("Error: The pull-Secret Path and public-Key Path must be provided. Running init interactive prompt")
			initialization(initFileName)
		}
		amiID, found := regions[*region]
		if !found {
			fmt.Println("Invalid or unsupported region:", *region)
			return
		}
		pullSecretPath, publicKeyPath = readPathsFromFile(initFileName)
		CAcertString, CAkeyString, err := createCertificateAuthority()
		if err != nil {
			fmt.Printf("Couldn't generate the CA cert and key with error: %v\n", err)
		}
		if len(*clusterVersion) > 0 {
			clusterFlag := true
			installRegistry(clusterFlag, pullSecretPath, publicKeyPath, *region, amiID, *clusterVersion, *openshiftCNI, *installConfigFlag, CAcertString, CAkeyString)
		} else {
			clusterFlag := false
			installRegistry(clusterFlag, pullSecretPath, publicKeyPath, *region, amiID, *clusterVersion, *openshiftCNI, *installConfigFlag, CAcertString, CAkeyString)
		}

		// If destroy flag is used destroy all
	} else if *destroyFlag && !*forceFlag {
		destroyRegistry()
		// If agent is down --force flag will simply destroy the mirror-registry host using raw terraform destroy command.
	} else if *destroyFlag && *forceFlag {
		fmt.Println("Destroying the infrastructure by running Terraform destroy command")
		mode := "destroy"
		terraformErr := runTerraform(mode)
		if terraformErr != nil {
			log.Fatalf("Failed to execute terraform destroy: %v", terraformErr)
			return
		}
		deleteGeneratedFiles()
	}
}

// This function executes the terraform command, Can be either apply or destroy.
func runTerraform(mode string) error {
	cmd := exec.Command("terraform", mode, "-auto-approve")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

// This is the main function that is being used to install the infrastructure requested by the user. Could be ONLY registry or also a cluster
func installRegistry(clusterFlag bool, pullSecretPath string, publicKeyPath string, region string, region_ami string, clusterVersion string, sdnCNI bool, installConfigFlag bool, CAcertString string, CAkeyString string) {

	// Create new PullSecretTemplate
	createPullSecretTemplate(pullSecretPath)
	// Update bash script with Pull Secret and Certs for the agent
	updateRegistryScriptFile(pullSecretTemplate, CAcertString, CAkeyString)
	// Replace the appropriate values in registry template terraform file
	UpdateCreateTfFileRegistry(publicKeyPath, region, region_ami)

	cmd := exec.Command("terraform", "init")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()

	mode := "apply"
	//Run the terraform apply command
	err := runTerraform(mode)
	if err != nil {
		log.Fatalf("Failed to execute terraform apply: %v", err)
	}

	// If there is a --cluster-version flag defined here we start the cluster installation. We contact the agent and agent is installing the cluster from the registry.
	if clusterFlag {
		fmt.Println("Sleeping for 5 minutes while waiting for the Registry and Agent to come up")
		time.Sleep(5 * time.Minute)
		applyTerraformConfig()
		for i := 1; i <= 10; i++ {
			GetInfraDetails()
			agentRegistryStatus := ClientGetStatus(infraDetailsStatus.InstancePublicDNS)
			if agentRegistryStatus && agentStatus.ClusterStatus == "DontExist" {
				installConfig := populateInstallConfigValues(sdnCNI, installConfigFlag)
				sendInstallConfigToAgent(installConfig, infraDetailsStatus.InstancePublicDNS)
				populateActionAndVersion(true, clusterVersion)
				sendActionAndVersionToAgent(infraDetailsStatus.InstancePublicDNS)
				break
			} else if agentRegistryStatus && agentStatus.ClusterStatus == "Exists" {
				fmt.Println("There is already a cluster deployed.")
				break
			} else if !agentRegistryStatus {
				fmt.Printf("Try No %v ... Registry is not yet ready. Retrying in 20 seconds", i)
			}
			if i == 10 {
				fmt.Println("The Registry is not up after 10 retries ( 8 minutes). There might be something wrong. Stop retrying")
			}
			time.Sleep(10 * time.Second)
		}
	} else {
		fmt.Println("No cluster version specified. Deploying only the registy.")
	}
}

// This fuction is to destroy the infrastructure. If agent is active first checks if there is a cluster there so to destroy this also.
func destroyRegistry() {
	GetInfraDetails()
	agentRegistryStatus := ClientGetStatus(infraDetailsStatus.InstancePublicDNS)
	if agentStatus.ClusterStatus == "Exists" {
		populateActionAndVersion(false, "")
		sendActionAndVersionToAgent(infraDetailsStatus.InstancePublicDNS)
	} else if agentRegistryStatus && agentStatus.ClusterStatus == "DontExist" {
		fmt.Println("Cluster does not exist. Destroying only the registry")
	}
	for i := 1; i <= 10; i++ {
		agentRegistryStatus := ClientGetStatus(infraDetailsStatus.InstancePublicDNS)
		if agentRegistryStatus {
			if agentStatus.ClusterStatus == "DontExist" {
				// Run the terraform destroy command
				fmt.Println("Destroying the infrastructure by running Terraform destroy command")
				mode := "destroy"
				terraformErr := runTerraform(mode)
				if terraformErr != nil {
					log.Fatalf("Failed to execute terraform destroy: %v", terraformErr)
					return
				}
				deleteGeneratedFiles()
				break
			} else if agentStatus.ClusterStatus == "Exists" {
				fmt.Printf("Try No %v... Cluster is still in destroying state, Re-checking in 2 minutes\n", i)
			}
			if i == 10 {
				fmt.Println("The Registry is not up after 10 retries ( 8 minutes). There might be something wrong. Stop retrying")
				break
			}
			time.Sleep(2 * time.Minute)
		}

	}
	fmt.Println("The infrastructure destroyed successfully")
}

// We update the registry initialization script "registry-mirror-script-terraform.sh.temp" and creates the "registry-mirror-script-terraform.sh.tpl"
func updateRegistryScriptFile(pullSecretTemp string, CAcert string, CAkey string) {
	pullSecretContent, err := os.ReadFile(pullSecretTemp)
	if err != nil {
		println("Cannot read the pull-secret")
	}
	pullSecretTemplateAsString := string(pullSecretContent)
	// Read registry template script file
	scriptContent, err := os.ReadFile(registryScriptTemplate)
	if err != nil {
		println("Cannot read the registry script template file")
	}

	addPullSecret := strings.ReplaceAll(string(scriptContent), "$PULL_SECRET_CONTENT$", pullSecretTemplateAsString)
	addCAcert := strings.ReplaceAll(string(addPullSecret), "$CA_CERT$", CAcert)
	addCAkey := strings.ReplaceAll(string(addCAcert), "$CA_KEY$", CAkey)

	error := os.WriteFile(registryScript, []byte(addCAkey), 0644)
	if error != nil {
		println("Cannot create the registry-script file")
	}
}

// To clean up the bash script, pull secret template, .tfvars and TF detail generated files after successfull deployment of the registry.
func deleteGeneratedFiles() {
	Script := os.Remove(registryScript)
	PullSecretTemp := os.Remove(pullSecretTemplate)
	os.Remove("terraform.tfvars")
	os.Remove("infra_details.json")
	os.Remove(CAcert)

	if Script != nil || PullSecretTemp != nil {
		return
	}
}

// It creates the pull Secret Template from the pull-secret.json provided by the user
func createPullSecretTemplate(pullSecret string) {

	serverToRemove := "cloud.openshift.com"
	newServer := "REGISTRY-HOSTNAME:8443"

	// Read the content of the pull secret file
	data, err := os.ReadFile(pullSecret)
	if err != nil {
		fmt.Printf("Failed to read the pullSecret file: %v\n", err)
		return
	}

	// Parse the JSON data into a map[string]interface{}
	var pullSecretMap map[string]interface{}
	if err := json.Unmarshal(data, &pullSecretMap); err != nil {
		fmt.Printf("Failed to unmarshal JSON data: %v\n", err)
		return
	}

	// Remove the specified server "cloud.openshift.com" for insights operator to not start.
	auths, authsExist := pullSecretMap["auths"].(map[string]interface{})
	if authsExist {
		delete(auths, serverToRemove)
	} else {
		fmt.Println("Error: 'auths' key not found in pull secret.")
		return
	}

	// Add the new server
	newAuth := map[string]interface{}{
		"auth":  "CREDENTIALS",
		"email": "registry@example.com",
	}
	auths[newServer] = newAuth

	// Convert the updated pullSecretMap back to JSON
	updatedData, err := json.Marshal(pullSecretMap)
	if err != nil {
		fmt.Println("Failed to marshal JSON data:", err)
		return
	}

	// Write the updated JSON data to a separate file
	err = os.WriteFile(pullSecretTemplate, updatedData, 0644)
	if err != nil {
		fmt.Println("Failed to create the updated pull-secret template file:", err)
		return
	}

	fmt.Println("Pull secret template updated successfully.")

}

// Here we populate the tfvars file with the infrastructure details before it is being used by Terraform.
func UpdateCreateTfFileRegistry(publicKey string, region string, amiID string) {

	// Read the contents of the Terraform template file
	fmt.Println("Updating and creating the Registry terraform file")
	templateContent, err := os.ReadFile("terraform.tfvars.temp")
	if err != nil {
		fmt.Println("Cannot read template file")
		return
	}

	availZoneA := (region + "a")
	availZoneB := (region + "b")
	availZoneC := (region + "c")

	// Replace the placeholder string with the generated public key path
	replacedPublicKey := strings.ReplaceAll(string(templateContent), "PUBLIC_KEY", publicKey)
	replacedRegion := strings.ReplaceAll(string(replacedPublicKey), "AWS_REGION", region)
	replacedAvailabilityZoneA := strings.ReplaceAll(string(replacedRegion), "AVAILABILITY_ZONE_A", availZoneA)
	replacedAvailabilityZoneB := strings.ReplaceAll(string(replacedAvailabilityZoneA), "AVAILABILITY_ZONE_B", availZoneB)
	replacedAvailabilityZoneC := strings.ReplaceAll(string(replacedAvailabilityZoneB), "AVAILABILITY_ZONE_C", availZoneC)
	updatedFile := strings.ReplaceAll(string(replacedAvailabilityZoneC), "AMI_ID", amiID)
	err = os.WriteFile("terraform.tfvars", []byte(updatedFile), 0644)
	if err != nil {
		fmt.Println("Cannot write the Terraform config file")
		return
	}
}

// Here we set the flag to the tfvars file in case there is a cluster installation required so we can provision all the cluster required resources.
func SetClusterFlagTerraform(flag bool) {
	// Read the contents of the Terraform template file
	fmt.Println("Creating the .tfvars file")
	templateContent, err := os.ReadFile("terraform.tfvars.temp")
	if err != nil {
		fmt.Println("Cannot read template file")
		return
	}
	if flag {
		flag_string := "true"
		// Set the cluster flag to true and create the terraform.tfvars file
		replacedClusterFlag := strings.ReplaceAll(string(templateContent), "false", flag_string)
		err = os.WriteFile("terraform.tfvars", []byte(replacedClusterFlag), 0644)
		if err != nil {
			fmt.Println("Cannot write the Terraform config file")
			return
		}

	} else if !flag {
		flag_string := "false"
		// Set the cluster flag to false and create the terraform.tfvars file
		replacedClusterFlag := strings.ReplaceAll(string(templateContent), "false", flag_string)
		err = os.WriteFile("terraform.tfvars", []byte(replacedClusterFlag), 0644)
		if err != nil {
			fmt.Println("Cannot write the Terraform config file")
			return
		}
	}
}

// Ask the user using shell prompt whatever is in question variable and return the input string.
func interactiveCLIFunction(question string) string {
	var s string
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprint(os.Stderr, question+" ")
		s, _ = r.ReadString('\n')
		if s != "" {
			break
		}
	}
	return strings.TrimSpace(s)
}

// Here we write the paths from the init command to the initData.json file.
func writePathsToFile(filename string, pathMap map[string]string) error {
	// Convert the map in JSON format.
	data, err := json.Marshal(pathMap)
	if err != nil {
		return err
	}

	// Create the json file so it is overriden each time the init is run
	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer file.Close()

	// Write the data in a json file
	_, err = file.Write(data)
	return err
}

// Here we read the paths from the init command from the initData.json file.
func readPathsFromFile(filename string) (pullSecret string, publickey string) {
	// Read the json file
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("Cannot Read init file")
	}
	// Return the contents in a map
	var pathMap map[string]string
	err = json.Unmarshal(data, &pathMap)
	if err != nil {
		fmt.Println("Cannot get pull-secret and public-key paths from init file")
	}

	pullSecretPathCurrent := pathMap["PullSecretPath"]
	publicKeyPathCurrent := pathMap["PublicKeyPath"]

	return pullSecretPathCurrent, publicKeyPathCurrent
}

// The whole process of getting the data and writing in the initData.json file
func getInitData(filepath string) {
	// Create a Map to store the paths provided
	pathMap := make(map[string]string)

	// Get the paths using interactive CLI
	pullSecretPathTemp := interactiveCLIFunction("Provide the absolute path of the pull-secret")

	// Check if the provided path is valid. If not run the initialization function
	if _, err := os.ReadFile(pullSecretPathTemp); err != nil {
		fmt.Printf("Failed to read the pullSecret file: %v\n", err)
		fmt.Println("Please provide a valid path")
		initialization(initFileName)
	}
	publicKeyPathTemp := interactiveCLIFunction("Provide the absolute path of the public key")

	// Check if the provided path is valid. If not run the initialization function
	if _, err := os.ReadFile(publicKeyPathTemp); err != nil {
		fmt.Printf("Failed to read the pullSecret file: %v\n", err)
		fmt.Println("Please provide a valid path")
		initialization(initFileName)
	}

	// Add paths and identifiers to the map
	pathMap["PullSecretPath"] = pullSecretPathTemp
	pathMap["PublicKeyPath"] = publicKeyPathTemp

	writePathsToFile(filepath, pathMap)
}

// Initializing the program and asks for required files
func initialization(initFile string) {
	deleteGeneratedFiles()
	getInitData(initFile)
	pullSecretPath, publicKeyPath = readPathsFromFile(initFile)
	fmt.Printf("Using pull-secret from file: %v\n", pullSecretPath)
	fmt.Printf("Using public-key from file: %v\n", publicKeyPath)
}

// Its being used as an additional way to check the provisioned infrastructure in case the agent is down. Its checking specific objects existence in the tfstate file.
func checkDeploymentState() (registyStatus bool, clusterStatus bool) {
	// Read the JSON file
	jsonData, err := os.ReadFile("./terraform.tfstate")
	if err != nil {
		fmt.Println("Probably there is no infrastructure provisioned or terraform.tfstate file is deleted/corrupted.")
		fmt.Println("Check if there is registry-mirror-script-terraform.tpl file under OCPD dir. If yes there might be orphan resources left to AWS")
		log.Fatal(err)

	}

	// Unmarshal the JSON into an empty interface (map[string]interface{})
	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		fmt.Println("Probably there is no infrastructure provisioned or terraform.tfstate file is deleted/corrupted.")
		fmt.Println("Check if there is registry-mirror-script-terraform.tpl file under OCPD dir. If yes there might be orphan resources left to AWS")
		log.Fatal(err)

	}

	// Check if "resources" key exists
	resources, resourcesExist := data["resources"].([]interface{})
	if !resourcesExist {
		log.Fatal("Error: 'resources' key is missing or not an array")
	}

	// Search for "aws_instance" type
	awsInstanceExists := false
	for _, resource := range resources {
		if res, ok := resource.(map[string]interface{}); ok {
			if resType, typeExist := res["type"].(string); typeExist && resType == "aws_instance" {
				awsInstanceExists = true
				break
			}
		}
	}

	// Search for "aws_iam_user" type
	enpointsExists := false
	for _, resource := range resources {
		if res, ok := resource.(map[string]interface{}); ok {
			if resType, typeExist := res["type"].(string); typeExist && resType == "aws_vpc_endpoint" {
				enpointsExists = true
				break
			}
		}
	}

	// Check what resources exist.
	if awsInstanceExists && enpointsExists {
		fmt.Println("There is infrastructure present.Already installed mirror registry and cluster")
		return true, true
	} else if awsInstanceExists {
		fmt.Println("There is infrastructure present. Already installed mirror registry")
		return true, false
	} else {
		fmt.Println("There is no infrastructure provisioned")
		return false, false
	}
}

// Here we define the struct that will hold the infrastructure details.

type InfraDetails struct {
	AWSRegion         string
	InstancePublicDNS string
	PrivateSubnet1    string
	PrivateSubnet2    string
	PrivateSubnet3    string
	PrivateDNS        string
	Token             string
}

// This functions gets the infrastructure ids from terraform and adds them in the struct InfraDetails for later use from the program
func GetInfraDetails() {

	initString := "terraform output --raw "

	region, err := GetTerraformOutputs(initString + "region")
	if err != nil {
		log.Fatalf("Failed to get region: %s\n", err)
	}

	instanceDNS, err := GetTerraformOutputs(initString + "ec2_instance_public_dns")
	if err != nil {
		log.Fatalf("Failed to get instanceId: %s\n", err)
	}

	subnet1ID, err := GetTerraformOutputs(initString + "private_subnet_1_id")
	if err != nil {
		log.Fatalf("Failed to get private subnet 1: %s\n", err)
	}

	subnet2ID, err := GetTerraformOutputs(initString + "private_subnet_2_id")
	if err != nil {
		log.Fatalf("Failed to get private subnet 2: %s\n", err)
	}

	subnet3ID, err := GetTerraformOutputs(initString + "private_subnet_3_id")
	if err != nil {
		log.Fatalf("Failed to get private subnet 3: %s\n", err)
	}

	ec2PrivateDNS, err := GetTerraformOutputs(initString + "ec2_private_hostname")
	if err != nil {
		log.Fatalf("Failed to get private DNS: %s\n", err)
	}

	randomToken, err := GetTerraformOutputs(initString + "random_token")
	if err != nil {
		log.Fatalf("Failed to get private DNS: %s\n", err)
	}

	infraDetailsStatus.AWSRegion = region
	infraDetailsStatus.InstancePublicDNS = instanceDNS
	infraDetailsStatus.PrivateSubnet1 = subnet1ID
	infraDetailsStatus.PrivateSubnet2 = subnet2ID
	infraDetailsStatus.PrivateSubnet3 = subnet3ID
	infraDetailsStatus.PrivateDNS = ec2PrivateDNS
	infraDetailsStatus.Token = randomToken
}

// Thats a helper for executing the terraform output commands
func GetTerraformOutputs(Cmd string) (string, error) {
	cmd := exec.Command("bash", "-c", Cmd)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
