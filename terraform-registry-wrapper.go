package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	//terraformTemplateFile  = "./Disconnected-template.tf.temp"
	terraformConfigFile    = "./Disconnected.tf"
	registryScriptTemplate = "./registry-mirror-script-terraform.sh.temp"
	registryScript         = "./registry-mirror-script-terraform.sh"
)

func main() {
	// Parse command-line arguments
	//region := flag.String("region", "", "Set the AWS region")
	installFlag := flag.Bool("install", false, "Install Registry")
	destroyFlag := flag.Bool("destroy", false, "Destroy Registry")
	privateFlag := flag.Bool("private", false, "Publish registry with private or public hostname")

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
	cmd.Dir = filepath.Dir(terraformConfigFile)
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
		// Update the Terraform configuration file with the user-provided region
		updateBashScript(private)
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
		deleteGeneratedScriptFile()
		return
	} else {
		mode := "destroy"
		// Run the terraform command
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
func deleteGeneratedScriptFile() {
	err := os.Remove(registryScript)
	if err != nil {
		// If an error occurs, print the error message
		fmt.Println("Error deleting file:", err)
		return
	}
}
