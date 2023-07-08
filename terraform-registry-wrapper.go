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
	//terraformTemplateFile  = "./Disconnected-template.tf.temp"
	terraformConfigFile    = "./Disconnected.tf"
	registryScriptTemplate = "./registry-mirror-script-terraform.sh.temp"
	registryScript         = "./registry-mirror-script-terraform.sh"
	pullSecret             = "./config/pull-secret.json"
	pullSecretTemplate     = "./pull-secret.template"
)

func main() {
	// Parse command-line arguments
	//region := flag.String("region", "", "Set the AWS region")
	installFlag := flag.Bool("install", false, "Install Registry")
	destroyFlag := flag.Bool("destroy", false, "Destroy Registry")
	privateFlag := flag.Bool("private", false, "Publish registry with private or public hostname. Default value False")

	flag.Parse()

	if *installFlag {
		install := true
		installOrDestroyRegistry(install, *privateFlag)
	} else if *destroyFlag {
		install := false
		installOrDestroyRegistry(install, *privateFlag)
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

func installOrDestroyRegistry(installFlag bool, private bool) {
	if installFlag {
		cmd := exec.Command("terraform", "init")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		cmd.Run()
		// Delete left over templates
		deleteGeneratedFiles()
		//Create new PullSecretTemplate
		createPullSecretTemplate(pullSecret)
		//Update the Bash Script with the provided information from the user.
		updateBashScript(private)
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

		return
	} else {
		mode := "destroy"
		// Run the terraform command
		deleteGeneratedFiles()
		err := runTerraform(mode)
		if err != nil {
			log.Fatalf("Failed to execute terraform destroy: %v", err)
		}
		return

	}
}

func updateBashScript(private bool) error {
	// Read the contents of the registry script file
	scriptContent, err := os.ReadFile(registryScriptTemplate)
	if err != nil {
		return err
	}
	if private {
		// Replace the placeholder string with the private hostname of the registry (for disconnected cluster deployments)
		changeHostnameSourceToFile := strings.ReplaceAll(string(scriptContent), "REGISTRY_URL", "hostname")
		err = os.WriteFile(registryScript, []byte(changeHostnameSourceToFile), 0644)
		//fmt.Println("changed the script to private")
		if err != nil {
			return err
		}
	} else {
		// Replace the placeholder string with the public hostname of the registry
		changeHostnameSourceToFile := strings.ReplaceAll(string(scriptContent), "REGISTRY_URL", "curl -s http://169.254.169.254/latest/meta-data/public-hostname")
		err = os.WriteFile(registryScript, []byte(changeHostnameSourceToFile), 0644)
		//fmt.Println("changed the script to public")
		if err != nil {
			return err
		}
	}
	return nil
}

// To clean up the bash script generated file after successfull deployment of the registry.
func deleteGeneratedFiles() {
	ScriptTemp := os.Remove(registryScript)
	PullSecretTemp := os.Remove(pullSecretTemplate)

	if ScriptTemp != nil || PullSecretTemp != nil {
		// If an error occurs, print the error message
		fmt.Println("one file is not deleted")
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
	}
}
