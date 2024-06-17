package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

//==========================================================================================
// Client type of functions
//==========================================================================================

func MonitoringDeployment(URL string) {

	var httpUrl string

	if !startsWithHTTP(URL) {
		httpUrl = "http://" + URL
	}

	fmt.Println(httpUrl)

	ClientGetStatus(httpUrl)

}

type InfraState struct {
	RegistryHealth string `json:"RegistryHealth"`
	ClusterStatus  string `json:"ClusterStatus"`
}

// Function to get the client status using HTTP. It expects a reply from the server-agent.
func ClientGetStatus(url string) bool {

	fmt.Println("ClientGetStatus started")

	resp, err := http.Get(url + ":8090/status")
	if err != nil {
		log.Println("Error making GET request:", err)
		time.Sleep(5 * time.Second)
		return false
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		time.Sleep(5 * time.Second)
		return false
	}

	s := &InfraState{}

	if err := json.Unmarshal(body, s); err != nil {
		fmt.Printf("error Unmarshaling JSON: %v\n", err)
		time.Sleep(5 * time.Second)
	}

	fmt.Println("InfraState: ", s)

	time.Sleep(5 * time.Second)

	return true
}

// To check if the URL is in appropriate format for the client
func startsWithHTTP(url string) bool {
	return len(url) >= 7 && (url[:7] == "http://" || len(url) >= 8 && url[:8] == "https://")
}

func sendClusterDetailsToServer(installconfig string, url string) {
	// Read the YAML file
	yamlFile, err := os.ReadFile(installconfig)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Convert the YAML to a generic map
	var yamlData map[string]interface{}
	err = yaml.Unmarshal(yamlFile, &yamlData)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Marshal the map into a JSON string
	jsonData, err := json.Marshal(yamlData)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Send the JSON data to the server
	resp, err := http.Post(url+":8090/deploy", "application/json", bytes.NewBuffer(jsonData))
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
