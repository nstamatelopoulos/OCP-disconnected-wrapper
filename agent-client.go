package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

//==========================================================================================
// Client type of functions
//==========================================================================================

func MonitoringDeployment(URL string) {

	fmt.Println(URL)

	ClientGetStatus(URL)

}

type InfraState struct {
	RegistryHealth string `json:"RegistryHealth"`
	ClusterStatus  string `json:"ClusterStatus"`
}

// Function to get the client status using HTTP. It expects a reply from the server-agent.
func ClientGetStatus(url string) bool {

	fmt.Println("ClientGetStatus started")

	if !startsWithHTTP(url) {
		url = "http://" + url
	}

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
