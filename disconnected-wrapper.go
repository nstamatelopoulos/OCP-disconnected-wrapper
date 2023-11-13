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
)

const (
	registryScriptTemplate = "./registry-mirror-script-terraform.sh.temp"
	registryScript         = "./registry-mirror-script-terraform.tpl"
	pullSecretTemplate     = "./pull-secret.template"
	initFileName           = "initData.json"
	currentStateFile       = "cluster.txt"
)

var regions = map[string]string{
	"eu-west-1":      "ami-0f0f1c02e5e4d9d9f",
	"eu-west-2":      "ami-035c5dc086849b5de",
	"eu-west-3":      "ami-0460bf124812bebfa",
	"eu-central-1":   "ami-0e7e134863fac4946",
	"eu-south-2":     "ami-031b6ef6108761a77",
	"eu-north-1":     "ami-06a2a41d455060f8b",
	"us-east-1":      "ami-06640050dc3f556bb",
	"us-east-2":      "ami-092b43193629811af",
	"us-west-1":      "ami-0186e3fec9b0283ee",
	"us-west-2":      "ami-08970fb2e5767e3b8",
	"ap-south-1":     "ami-05c8ca4485f8b138a",
	"ap-northeast-3": "ami-044921b7897a7e0da",
	"ap-northeast-2": "ami-06c568b08b5a431d5",
	"ap-southeast-1": "ami-051f0947e420652a9",
	"ap-southeast-2": "ami-0808460885ff81045",
	"ap-northeast-1": "ami-0f903fb156f24adbf",
	"ca-central":     "ami-0c3d3a230b9668c02",
	"sa-east-1":      "ami-0c1b8b886626f940c",
}
var pullSecretPath string
var publicKeyPath string

func main() {

	// Parse command-line arguments
	region := flag.String("region", "", "Set the AWS region")
	installFlag := flag.Bool("install", false, "Install Registry")
	destroyFlag := flag.Bool("destroy", false, "Destroy Registry")
	clusterVersion := flag.String("cluster-version", "", "Set the prefered cluster version")
	initFlag := flag.Bool("init", false, "Saving pull-secret and public-key for ease of use")
	helpFlag := flag.Bool("help", false, "Help")

	flag.Parse()

	consolidatedFlagCheckFunction(*installFlag, *destroyFlag, *region, *clusterVersion, *initFlag, *helpFlag)

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
			checkDeploymentState()
		} else {
			fmt.Println("Error:", err)
		}

		// Delete left over templates
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
		if len(*clusterVersion) > 0 {
			clusterFlag := true
			installRegistry(clusterFlag, pullSecretPath, publicKeyPath, *region, amiID, *clusterVersion)
		} else {
			clusterFlag := false
			installRegistry(clusterFlag, pullSecretPath, publicKeyPath, *region, amiID, *clusterVersion)
		}

		// If destroy flag is used destroy all
	} else if *destroyFlag {
		cluster_destroyed := interactiveCLIFunction("Did you destroy the cluster (yes|no)")
		if cluster_destroyed == "no" {
			fmt.Println("Please destroy the cluster before you destroy the registry")
			os.Exit(1)
		} else if cluster_destroyed == "yes" {
			fmt.Println("Destroying the registry")
		} else {
			fmt.Println("No valid answer. Please type yes or no")
			return
		}
		destroyRegistry()
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

func installRegistry(clusterFlag bool, pullSecretPath string, publicKeyPath string, region string, region_ami string, clusterVersion string) {

	// Create new PullSecretTemplate
	createPullSecretTemplate(pullSecretPath)
	// Update the Bash Script with the provided information from the user.
	updateBashScript(clusterFlag, clusterVersion)
	// If cluster flag is used set it to true else set to false in terraform.tfstate file and created it.
	SetClusterFlagTerraform(clusterFlag)
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

}

func destroyRegistry() {
	// Run the terraform destroy command
	mode := "destroy"
	err := runTerraform(mode)
	if err != nil {
		log.Fatalf("Failed to execute terraform destroy: %v", err)
		return
	}
	deleteGeneratedFiles()
}

// The updateBashScript function is changes the variables of the bash script template and writes it in a new file.
func updateBashScript(private bool, clusterVersion string) {
	// Set hostname to the mirror-registry
	pullSecretContent, err := os.ReadFile(pullSecretTemplate)
	if err != nil {
		println("Cannot read the pull-secret")
	}
	pullSecretTemplateAsString := string(pullSecretContent)
	// Read registry template script file
	scriptContent, err := os.ReadFile(registryScriptTemplate)
	if err != nil {
		println("Cannot read the registry script template file")
	}
	// Update the variables with their value
	addPullSecret := strings.ReplaceAll(string(scriptContent), "$PULL_SECRET_CONTENT$", pullSecretTemplateAsString)
	// If the private flag is true add the cluster variable in the registry script
	if private {
		addClusterFlag := strings.ReplaceAll(addPullSecret, "CREATE_CLUSTER=false", "CREATE_CLUSTER=true")
		setClusterVersionVar := strings.ReplaceAll(addClusterFlag, "$PICK_A_VERSION$", clusterVersion)
		// Create the Release channel from the cluster version provided from the user
		parts := strings.Split(clusterVersion, ".")
		if len(parts) >= 2 {
			// Take the first two parts and concatenate "stable-" in front of them
			clusterReleaseChannnel := "stable-" + parts[0] + "." + parts[1]
			setReleaseChannel := strings.ReplaceAll(setClusterVersionVar, "$PICK_A_CHANNEL$", clusterReleaseChannnel)

			withCluster := os.WriteFile(registryScript, []byte(setReleaseChannel), 0644)
			if withCluster != nil {
				println("Cannot write the cluster variable to the registry script file")
			}
		}
		// If the private flag is not true then simply write the file with the default changes
	} else {
		withoutCluster := os.WriteFile(registryScript, []byte(addPullSecret), 0644)
		if withoutCluster != nil {
			println("Cannot write the pull-secret to the registry script file")
		}
	}
}

// To clean up the bash script, pull secret template and .tfvars generated files after successfull deployment of the registry.
func deleteGeneratedFiles() {
	Script := os.Remove(registryScript)
	PullSecretTemp := os.Remove(pullSecretTemplate)
	os.Remove("terraform.tfvars")

	if Script != nil || PullSecretTemp != nil {
		return
	}
}

// It creates the pull Secret Template from the pull-secret.json provided by the user
func createPullSecretTemplate(pullSecret string) {
	// convert the string file to []byte and add it to data variable
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
	json.Unmarshal(data, &pullSecretMap)

	// Add the new section under "auths"
	auths := pullSecretMap["auths"].(map[string]interface{})
	newAuth := map[string]interface{}{
		"auth":  "CREDENTIALS",
		"email": "registry@example.com",
	}
	auths["REGISTRY-HOSTNAME:8443"] = newAuth

	// Convert the updated pullSecretMap back to JSON
	updatedData, _ := json.MarshalIndent(pullSecretMap, "", "  ")

	// Write the updated JSON data to a separate file
	error := os.WriteFile(pullSecretTemplate, updatedData, 0644)
	if error != nil {
		fmt.Println("Failed to create the pull-secret template file")
		return
	}
}

func UpdateCreateTfFileRegistry(publicKey string, region string, amiID string) {

	// Read the contents of the Terraform template file
	fmt.Println("Updating and creating the Registry terraform file")
	templateContent, err := os.ReadFile("terraform.tfvars")
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

func checkDeploymentState() {
	// Read the JSON file
	jsonData, err := os.ReadFile("./terraform.tfstate")
	if err != nil {
		log.Fatal(err)
	}

	// Unmarshal the JSON into an empty interface (map[string]interface{})
	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
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
	awsUserExists := false
	for _, resource := range resources {
		if res, ok := resource.(map[string]interface{}); ok {
			if resType, typeExist := res["type"].(string); typeExist && resType == "aws_iam_user" {
				awsUserExists = true
				break
			}
		}
	}

	// Check what resources exist.
	if awsInstanceExists && awsUserExists {
		fmt.Println("There is already infrastructure present. You cannot deploy new infrastructure before destroy the current one")
		clusterInfo := "Already installed mirror registry and cluster"
		fmt.Println(clusterInfo)
		os.Exit(1)
	} else if awsInstanceExists {
		fmt.Println("There is already infrastructure present. You cannot deploy new infrastructure before destroy the current one")
		clusterInfo := "Already installed mirror registry"
		fmt.Println(clusterInfo)
		os.Exit(1)
	}

}
