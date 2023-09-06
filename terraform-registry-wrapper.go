package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	cluster_TF             = "./Build_Cluster_Dependencies.tf"
	registry_TF            = "./Build_Registry.tf"
	registryScriptTemplate = "./registry-mirror-script-terraform.sh.temp"
	registryScript         = "./registry-mirror-script-terraform.tpl"
	pullSecretTemplate     = "./pull-secret.template"
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

func main() {

	// Parse command-line arguments
	region := flag.String("region", "", "Set the AWS region")
	installFlag := flag.Bool("install", false, "Install Registry")
	destroyFlag := flag.Bool("destroy", false, "Destroy Registry")
	clusterFlag := flag.Bool("cluster", false, "Create a disconnected cluster. Default value False")
	pullSecretPath := flag.String("pull-secret", "", "Set the Path to the user provided pull-secret")
	publicKeyPath := flag.String("public-key", "", "Set the path to the user public key")

	flag.Parse()

	if *installFlag {
		amiID, found := regions[*region]
		if !found {
			fmt.Println("Invalid or unsupported region:", *region)
			return
		}

		installRegistry(*clusterFlag, *pullSecretPath, *publicKeyPath, *region, amiID)
	} else if *destroyFlag {
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

func installRegistry(clusterFlag bool, pullSecretPath string, publicKeyPath string, region string, region_ami string) {

	cmd := exec.Command("terraform", "init")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()
	// Delete left over templates
	deleteGeneratedFiles()
	//Create new PullSecretTemplate
	createPullSecretTemplate(pullSecretPath)
	//Update the Bash Script with the provided information from the user.
	updateBashScript(clusterFlag)
	//Replace the appropriate values in registry template terraform file
	UpdateCreateTfFileRegistry(publicKeyPath, region, region_ami)
	//If cluster flag is used replace the appropriate values in cluster dependencies terraform file
	if clusterFlag {
		UpdateCreateTfFileCluster(region)
		//CombineTemporaryFiles()
	}
	mode := "apply"
	// Run the terraform command
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
	os.Remove(cluster_TF)
	os.Remove(registry_TF)
	deleteGeneratedFiles()
}

// The updateBashScript function is changes the variables of the bash script template and writes it in a new file.
func updateBashScript(private bool) {
	var urlString string
	if private {
		urlString = "hostname"
	} else {
		urlString = "curl -s http://169.254.169.254/latest/meta-data/public-hostname"
	}
	pullSecretContent, err := os.ReadFile(pullSecretTemplate)
	if err != nil {
		println("Cannot read the pull-secret")
	}
	pullSecretTemplateAsString := string(pullSecretContent)
	//Map of the replacement strings in the script file
	replacements := map[string]string{
		"$PULL_SECRET_CONTENT$": pullSecretTemplateAsString,
		"$REGISTRY_URL$":        urlString,
	}
	scriptContent, err := os.ReadFile(registryScriptTemplate)
	if err != nil {
		println("Cannot read the registry script template file")
	}
	scriptContentAsString := string(scriptContent)
	for oldString, newString := range replacements {
		scriptContentAsString = strings.ReplaceAll(scriptContentAsString, oldString, newString)
	}
	error := os.WriteFile(registryScript, []byte(scriptContentAsString), 0644)
	if error != nil {
		println("Cannot write the changes to the registry script file")
	}
}

// To clean up the bash script generated file after successfull deployment of the registry.
func deleteGeneratedFiles() {
	Script := os.Remove(registryScript)
	PullSecretTemp := os.Remove(pullSecretTemplate)

	if Script != nil || PullSecretTemp != nil {
		// If an error occurs, print the error message
		//fmt.Println("one file is not deleted")
		return
	}
}

// It creates the pull Secret Template from the pull-secret.json provided by the user
func createPullSecretTemplate(pullSecret string) {
	// convert the string file to []byte and add it to data variable
	data, _ := os.ReadFile(pullSecret)

	// Parse the JSON data into a map[string]interface{}
	var pullSecretMap map[string]interface{}
	json.Unmarshal(data, &pullSecretMap)

	// Add the new section under "auths"
	auths := pullSecretMap["auths"].(map[string]interface{})
	newAuth := map[string]interface{}{
		"auth":  "CREDENTIALS",
		"email": "registry@example.com",
	}
	auths["REGISTRY-HOSTNAME"] = newAuth

	// Convert the updated pullSecretMap back to JSON
	updatedData, _ := json.MarshalIndent(pullSecretMap, "", "  ")

	// Write the updated JSON data to a separate file
	err := os.WriteFile(pullSecretTemplate, updatedData, 0644)
	if err != nil {
		fmt.Println("Failed to create the pull-secret template file")
		return
	}
}

func UpdateCreateTfFileRegistry(publicKey string, region string, amiID string) {

	// Read the contents of the Terraform template file
	fmt.Println("Updating and creating the Registry terraform file")
	templateContent, err := os.ReadFile("Disconnected-template.tf.temp")
	if err != nil {
		fmt.Println("Cannot read template file")
		return
	}

	availZoneA := (region + "a")
	availZoneB := (region + "b")
	availZoneC := (region + "c")

	// Replace the placeholder string with the generated public key path
	replacedPublicKey := strings.ReplaceAll(string(templateContent), "PUBLIC_KEY_PATH", publicKey)
	replacedRegion := strings.ReplaceAll(string(replacedPublicKey), "AWS_REGION", region)
	replacedAvailabilityZoneA := strings.ReplaceAll(string(replacedRegion), "AVAILABILITY_ZONE_A", availZoneA)
	replacedAvailabilityZoneB := strings.ReplaceAll(string(replacedAvailabilityZoneA), "AVAILABILITY_ZONE_B", availZoneB)
	replacedAvailabilityZoneC := strings.ReplaceAll(string(replacedAvailabilityZoneB), "AVAILABILITY_ZONE_C", availZoneC)
	updatedFile := strings.ReplaceAll(string(replacedAvailabilityZoneC), "AMI_ID", amiID)
	err = os.WriteFile("Build_Registry.tf", []byte(updatedFile), 0644)
	if err != nil {
		fmt.Println("Cannot write the Terraform config file")
		return
	}
}

func UpdateCreateTfFileCluster(region string) {
	// Read the contents of the Terraform template file
	fmt.Println("Updating and creating the Cluster dependencies file")
	templateContent, err := os.ReadFile("cluster-dependencies.tf.temp")
	if err != nil {
		fmt.Println("Cannot read template file")
		return
	}

	// Replace the placeholder string with the generated public key path
	replacedRegion := strings.ReplaceAll(string(templateContent), "AWS_REGION", region)
	err = os.WriteFile("Build_Cluster_Dependencies.tf", []byte(replacedRegion), 0644)
	if err != nil {
		fmt.Println("Cannot write the Terraform config file")
		return
	}
}
