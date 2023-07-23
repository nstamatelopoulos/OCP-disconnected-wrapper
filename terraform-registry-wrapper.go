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
	terraformTemplateFile  = "./Disconnected-template.tf.temp"
	terraformConfigFile    = "./Disconnected.tf"
	registryScriptTemplate = "./registry-mirror-script-terraform.sh.temp"
	registryScript         = "./registry-mirror-script-terraform.sh"
	pullSecretTemplate     = "./pull-secret.template"
)

func main() {
	// Parse command-line arguments
	//region := flag.String("region", "", "Set the AWS region")   // To be enabled in the future. Pending to create an mapping between AMI IDs and regions of the mirror registy instance image
	installFlag := flag.Bool("install", false, "Install Registry")
	destroyFlag := flag.Bool("destroy", false, "Destroy Registry")
	privateFlag := flag.Bool("private", false, "Publish registry with private or public hostname. Default value False")
	pullSecretPath := flag.String("pull-secret", "", "Set the Path to the user provided pull-secret")
	publicKeyPath := flag.String("public-key", "", "Set the path to the user public key")

	flag.Parse()

	if *installFlag {
		install := true
		installRegistry(install, *privateFlag, *pullSecretPath, *publicKeyPath)
	} else if *destroyFlag {
		install := false
		destroyRegistry(install, *privateFlag)
	}

}

/*func updateTerraformConfig(region string) error {
	// Read the contents of the Terraform template file
	templateContent, err := os.ReadFile(terraformTemplateFile)
	if err != nil {
		return err
	}
	// Replace the placeholder string with the user-provided region
	changeRegionToFile := strings.ReplaceAll(string(templateContent), "Region-Value", region)
	// Replace the Availability Zone according to the region provided
	var availabilityZone string = region + "a"
	changeAvailZoneToFile := strings.ReplaceAll(changeRegionToFile, "Availability-Zone", availabilityZone)
	// Write the updated content to the Terraform configuration file
	err = os.WriteFile(terraformConfigFile, []byte(changeAvailZoneToFile), 0644)
	if err != nil {
		return err
	}

	return nil
}*/

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

func installRegistry(installFlag bool, private bool, pullSecretPath string, publicKeyPath string) {

	cmd := exec.Command("terraform", "init")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()
	// Delete left over templates
	deleteGeneratedFiles()
	//Create new PullSecretTemplate
	createPullSecretTemplate(pullSecretPath)
	//Update the Bash Script with the provided information from the user.
	updateBashScript(private)
	//Import the SSH public and private key to the terraform file to be used from instance creation and file provisioners.
	importSSHKeyToTerraformfile(publicKeyPath)
	// Update the region in the terraform file.
	/*if region != "" {
		err := updateTerraformConfig(region)
		if err != nil {
			log.Fatalf("Failed to update Terraform configuration: %v", err)
		}
	} else {
		log.Fatal("Region not provided. Please specify the --region flag.")
	}*/
	mode := "apply"
	// Run the terraform command
	err := runTerraform(mode)
	if err != nil {
		log.Fatalf("Failed to execute terraform apply: %v", err)
	}
}

func destroyRegistry(installFlag bool, private bool) {
	// Run the terraform destroy command
	mode := "destroy"
	err := runTerraform(mode)
	if err != nil {
		log.Fatalf("Failed to execute terraform destroy: %v", err)
		return
	}
	os.Remove(terraformConfigFile)
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

func importSSHKeyToTerraformfile(publicKey string) {
	// Read the contents of the Terraform template file
	templateContent, err := os.ReadFile(terraformTemplateFile)
	if err != nil {
		fmt.Println("Cannot read template file")
		return
	}
	// Replace the placeholder string with the generated public key path
	addPublicKeyPath := strings.ReplaceAll(string(templateContent), "PUBLIC_KEY_PATH", publicKey)
	err = os.WriteFile(terraformConfigFile, []byte(addPublicKeyPath), 0644)
	if err != nil {
		fmt.Println("Cannot write the Terraform config file")
		return
	}
}
