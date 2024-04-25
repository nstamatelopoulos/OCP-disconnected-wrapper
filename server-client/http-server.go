package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	url        = "https://localhost:8443"
	installDir = "/ocpd"
)

var (
	isRegistryHealthy         bool
	isClusterInstalled        bool
	healthMutex, clusterMutex sync.Mutex
)

type InfraStatus struct {
	RegistryHealth string
	ClusterStatus  string
}

func main() {

	go monitorRegistry(url)

	go monitorClusterInstallation(installDir)

	agentHTTPServer()

	select {}
}

func agentHTTPServer() {

	// Here we reply with the status of Registry and Cluster
	status := &InfraStatus{}

	http.HandleFunc("/status", func(w http.ResponseWriter, req *http.Request) {
		status.statusHandler(w)
	})

	fmt.Println("Starting HTTP Agent")
	if err := http.ListenAndServe(":8090", nil); err != nil {
		fmt.Printf("Error Starting HTTP Agent: %s\n", err)
	}

}

func (s *InfraStatus) setDataInStruct(getHealthStatus bool, getClusterStatus bool) {

	var registry, cluster string

	if getHealthStatus {
		registry = "Healthy"
	} else {
		registry = "Unknown"
	}

	if getClusterStatus {
		cluster = "Exists"
	} else {
		cluster = "Unknown"
	}

	s.RegistryHealth = registry
	s.ClusterStatus = cluster

}

// Provide the status of the infrastructure as a json http response
func (s *InfraStatus) statusHandler(w http.ResponseWriter) {

	// Set the response content type to json
	w.Header().Set("Content-Type", "application/json")

	s.setDataInStruct(getHealthStatus(), getClusterStatus())

	jsonData, err := json.Marshal(s)
	if err != nil {
		// If there is an error in marshaling, send an HTTP error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write the JSON data to the response
	w.Write(jsonData)
}

// Monitors the Registry by testing port 8443 every 5 seconds
func monitorRegistry(url string) {

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	for {

		fmt.Println("Monitoring remote port...")
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Error making GET request: %s\n", err)
			setHealthStatus(false)
		} else {

			if resp.StatusCode == http.StatusOK {
				fmt.Printf("The registry listens to %s\n", url)
				setHealthStatus(true)
			} else if resp.StatusCode != http.StatusOK {
				fmt.Printf("Received non-200 status code: %d\n", resp.StatusCode)
				setHealthStatus(false)
			}
			resp.Body.Close()
		}
		time.Sleep(5 * time.Second)
	}
}

func setHealthStatus(health bool) {
	healthMutex.Lock()
	isRegistryHealthy = health
	healthMutex.Unlock()
}

func getHealthStatus() bool {
	healthMutex.Lock()
	defer healthMutex.Unlock()
	return isRegistryHealthy
}

// Here we monitor the if the cluster installation is present. We check that by checking for a terraform.tfstate files in the installation directory.
// We check every 5 seconds.
func monitorClusterInstallation(installDir string) {

	for {
		bootstrapExists := false
		clusterExists := false

		bootstrapFile := installDir + "/" + "terraform.bootstrap.tfstate"
		clusterFile := installDir + "/" + "terraform.cluster.tfstate"

		fmt.Printf("The path is: %s\n", bootstrapFile)
		fmt.Printf("The path is: %s\n", clusterFile)

		if _, err := os.Stat(bootstrapFile); os.IsNotExist(err) {
			fmt.Println("No terraform.bootstrap.tfstate file detected.")
		} else if err == nil {
			bootstrapExists = true
			fmt.Println("Terraform.bootstrap.tfstate file detected.")
		}

		if _, err := os.Stat(clusterFile); os.IsNotExist(err) {
			fmt.Println("No terraform.cluster.tfstate file detected.")
		} else if err == nil {
			clusterExists = true
			fmt.Println("Terraform.cluster.tfstate file detected.")
		}

		if bootstrapExists || clusterExists {
			fmt.Println("At least one Terraform tfstate file detected. There is a cluster installation present")
			setClusterStatus(true)
		} else {
			setClusterStatus(false)
		}

		time.Sleep(5 * time.Second)
	}
}

func setClusterStatus(clusterState bool) {
	clusterMutex.Lock()
	isClusterInstalled = clusterState
	clusterMutex.Unlock()
}

func getClusterStatus() bool {
	clusterMutex.Lock()
	defer clusterMutex.Unlock()
	return isClusterInstalled
}
