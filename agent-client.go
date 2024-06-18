package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

//==========================================================================================
// Client type of functions
//==========================================================================================

func MonitoringDeployment(url string) (string, string) {

	if !startsWithHTTP(url) {
		url = "http://" + url
	}

	fmt.Println(url)

	registryHealth, clusterStatus := ClientGetStatus(url)

	return registryHealth, clusterStatus

}

type InfraState struct {
	RegistryHealth string `json:"RegistryHealth"`
	ClusterStatus  string `json:"ClusterStatus"`
}

// Function to get the client status using HTTP. It expects a reply from the server-agent.
func ClientGetStatus(url string) (string, string) {

	fmt.Println("ClientGetStatus started")

	resp, err := http.Get(url + ":8090/status")
	if err != nil {
		log.Println("Error making GET request:", err)
		time.Sleep(5 * time.Second)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		time.Sleep(5 * time.Second)
	}

	s := &InfraState{}

	if err := json.Unmarshal(body, s); err != nil {
		fmt.Printf("error Unmarshaling JSON: %v\n", err)
		time.Sleep(5 * time.Second)
	}

	fmt.Println("InfraState: ", s)

	return s.RegistryHealth, s.ClusterStatus
}

// To check if the URL is in appropriate format for the client
func startsWithHTTP(url string) bool {
	return len(url) >= 7 && (url[:7] == "http://" || len(url) >= 8 && url[:8] == "https://")
}

func sendClusterDetailsToAgent(installconfig string, url string) {
	// Read the YAML file
	fmt.Println("Reading Install Config from path:", installconfig)
	yamlFile, err := os.ReadFile(installconfig)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Convert the YAML to a generic map
	fmt.Println("Converting to map")
	var yamlData map[string]interface{}
	err = yaml.Unmarshal(yamlFile, &yamlData)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Marshal the map into a JSON string
	fmt.Println("Marshaling the map to json")
	jsonData, err := json.Marshal(yamlData)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Send the JSON data to the server
	fmt.Println("Sending the data using Post request")
	resp, err := http.Post("http://"+url+":8090/deploy", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	defer resp.Body.Close()

	// Print response from server
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	log.Println("Response from server:", string(body))
}

func installCluster(installConfig string) {
	//======================================================================================
	//The below is the Terraform part of the infrastructure. Not the openshift-install part.
	//======================================================================================

	// Read the contents of the Terraform template file
	fmt.Println("Updating .tfvars file with cluster flag")
	templateContent, err := os.ReadFile("terraform.tfvars")
	if err != nil {
		fmt.Println("Cannot read template file")
		return
	}

	// Set the cluster flag to true and create the terraform.tfvars file
	replacedClusterFlag := strings.ReplaceAll(string(templateContent), "false", "true")
	err = os.WriteFile("terraform.tfvars", []byte(replacedClusterFlag), 0644)
	if err != nil {
		fmt.Println("Cannot write the Terraform config file")
		return
	}

	cmd := exec.Command("terraform", "apply", "-target=module.Cluster_Dependencies", "-auto-approve")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	error := cmd.Run()
	if error != nil {
		fmt.Printf("Terraform apply failed with: %v", error)
	}

	// Wait some seconds for terraform to get applied
	time.Sleep(5 * time.Second)

	//======================================================================================
	//The below is the openshift-install part.
	//======================================================================================

	Ec2UrlRaw := GetInfraDetails("InstancePublicDNS")

	registryHealth, clusterStatus := MonitoringDeployment(Ec2UrlRaw)

	if registryHealth == "Healthy" && clusterStatus == "DontExist" {
		sendClusterDetailsToAgent(installConfig, Ec2UrlRaw)
	}
}
