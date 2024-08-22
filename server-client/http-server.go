package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	url        = "https://localhost:8443"
	installDir = "/ec2-user/cluster"
)

var (
	isRegistryHealthy         bool
	isClusterInstalled        bool
	healthMutex, clusterMutex sync.Mutex
	status                    *InfraStatus
	statusMutex               sync.Mutex
	agentAction               *DeployDestroy
)

type InfraStatus struct {
	RegistryHealth string
	ClusterStatus  string
}

type DeployDestroy struct {
	ClusterVersion string
	Deploy         string
}

func main() {

	status = &InfraStatus{}

	agentAction = &DeployDestroy{}

	// Open a file for logging
	logFile, err := os.OpenFile("/app/monitoring.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	fmt.Println("Starting monitoring the deployment")

	go monitorRegistry(url)

	go monitorClusterInstallation(installDir)

	agentHTTPServer()

	select {}
}

func agentHTTPServer() {

	fmt.Println("Starting HTTP agent-server")

	// Here we reply with the status of Registry and Cluster

	http.HandleFunc("/status", func(w http.ResponseWriter, req *http.Request) {
		statusHandler(w)
	})

	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		installConfigHandler(w, r)
	})

	http.HandleFunc("/action", deployDestroyHandler)

	fmt.Println("Starting HTTP Agent")
	if err := http.ListenAndServe(":8090", nil); err != nil {
		fmt.Printf("Error Starting HTTP Agent: %s\n", err)
	}

}

func updateInfraStatus() {
	statusMutex.Lock()
	defer statusMutex.Unlock()

	registryHealth := "Unhealthy"
	if getHealthStatus() {
		registryHealth = "Healthy"
	}

	clusterStatus := "DontExist"
	if getClusterStatus() {
		clusterStatus = "Exists"
	}

	status.RegistryHealth = registryHealth
	status.ClusterStatus = clusterStatus
}

// ======================================================================================
// This is the HTTP handler for requests comming on path /status
// ======================================================================================
func statusHandler(w http.ResponseWriter) {

	// Set the response content type to json
	w.Header().Set("Content-Type", "application/json")

	statusMutex.Lock()
	defer statusMutex.Unlock()

	jsonData, err := json.Marshal(status)
	if err != nil {
		// If there is an error in marshaling, send an HTTP error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write the JSON data to the response
	w.Write(jsonData)
}

//======================================================================================
//This is the HTTP handler for requests comming on path /deploy
//======================================================================================

func installConfigHandler(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	fmt.Println("Using deployHandler")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Unmarshal the JSON data to a generic map
	fmt.Println("Unmarshal the JSON")
	var jsonData map[string]interface{}
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		fmt.Println("Unmarshal the JSON error:", err)
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}

	// Marshal the map back to YAML
	fmt.Println("Marshal the JSON to yaml")
	yamlData, err := yaml.Marshal(jsonData)
	if err != nil {
		fmt.Println("Marshal the JSON to yaml:", err)
		http.Error(w, "Error converting to YAML", http.StatusInternalServerError)
		return
	}

	// Save the YAML data to a file
	fmt.Println("Save the YAML data to a file")
	filePath := installDir + "/install-config.yaml"

	fmt.Println("The file path is:", filePath)
	err = os.WriteFile(filePath, yamlData, 0644)
	if err != nil {
		http.Error(w, "Error writing file", http.StatusInternalServerError)
		fmt.Println("Save the YAML data to a file error is:", err)
		return
	}

	// Respond to the client
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Data received and saved successfully"))
}

//======================================================================================
// Monitors the Registry by testing port 8443 every 5 seconds
//======================================================================================

func monitorRegistry(url string) {

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	for {

		log.Println("Monitoring remote port...")
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Error making GET request: %s\n", err)
			setHealthStatus(false)
		} else {

			if resp.StatusCode == http.StatusOK {
				log.Printf("The registry listens to %s\n", url)
				setHealthStatus(true)
			} else if resp.StatusCode != http.StatusOK {
				log.Printf("Received non-200 status code: %d\n", resp.StatusCode)
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
	updateInfraStatus()
}

func getHealthStatus() bool {
	healthMutex.Lock()
	defer healthMutex.Unlock()
	return isRegistryHealthy
}

//======================================================================================
// Here we monitor the if the cluster installation is present. We check that by checking for a terraform.tfstate files in the installation directory.
// We check every 5 seconds.
//======================================================================================

func monitorClusterInstallation(installDir string) {

	for {
		bootstrapExists := false
		clusterExists := false

		bootstrapFile := installDir + "/" + "terraform.bootstrap.tfstate"
		clusterFile := installDir + "/" + "terraform.cluster.tfstate"

		log.Printf("The path is: %s\n", bootstrapFile)
		log.Printf("The path is: %s\n", clusterFile)

		if _, err := os.Stat(bootstrapFile); os.IsNotExist(err) {
			log.Println("No terraform.bootstrap.tfstate file detected.")
		} else if err == nil {
			bootstrapExists = true
			log.Println("Terraform.bootstrap.tfstate file detected.")
		}

		if _, err := os.Stat(clusterFile); os.IsNotExist(err) {
			log.Println("No terraform.cluster.tfstate file detected.")
		} else if err == nil {
			clusterExists = true
			log.Println("Terraform.cluster.tfstate file detected.")
		}

		if bootstrapExists || clusterExists {
			log.Println("At least one Terraform tfstate file detected. There is a cluster installation present")
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
	updateInfraStatus()
}

func getClusterStatus() bool {
	clusterMutex.Lock()
	defer clusterMutex.Unlock()
	return isClusterInstalled
}

// ======================================================================================
// This is to start the openshift installation using the OCP installer
// ======================================================================================

func installOrDestroyCluster(action string, clusterVersion string) {

	if agentAction.Deploy == "Install" && len(agentAction.ClusterVersion) > 0 {
		populateVersionToInstallerScript(clusterVersion)
		installCluster()
	} else if agentAction.Deploy == "Destroy" && len(agentAction.ClusterVersion) == 3 {
		fmt.Println("Destroying cluster")
		destroyCluster()
	} else {
		fmt.Printf("Invalid combination of actionAndVersion. Action: %s Version: %s", action, clusterVersion)
	}

}

func destroyCluster() {

	fmt.Println("Running the openshift-install destroy command")

	cmdStr := `echo 'export PATH="/ec2-user/bin:$PATH"' >> $HOME/.bashrc && \
	echo 'export AWS_SHARED_CREDENTIALS_FILE=/ec2-user/.aws/credentials' >> $HOME/.bashrc && \
	source $HOME/.bashrc && \
	openshift-install destroy cluster --dir "` + installDir + `" --log-level debug && \
	rm -rf /ec2-user/bin/openshift-install && \
	rm -rf /ec2-user/bin/oc && \
	rm -rf /ec2-user/mirroring-workspace/imageset-config.yaml && \
	rm -rf /app/cluster-installation-script.sh && \
	rm -rf /ec2-user/cluster/.openshift_install.log`

	cmd := exec.Command("bash", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the command and check for errors
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running openshift-install destroy command: %v\n", err)
	} else {
		fmt.Println("openshift-install destroy executed successfully")
	}
}

func installCluster() {
	fmt.Println("Running the installation script as ec2-user")

	cmdStr := `chmod +x /app/cluster-installation-script.sh && /app/cluster-installation-script.sh`

	cmd := exec.Command("bash", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the command and check for errors
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running script: %v\n", err)
	} else {
		fmt.Println("Script executed successfully")
	}

}

func deployDestroyHandler(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	fmt.Println("Using desployDestroyHandler")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read actionForAgent request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Unmarshal the JSON data to a generic map
	fmt.Println("Unmarshal the JSON")
	err = json.Unmarshal(body, &agentAction)
	if err != nil {
		fmt.Println("Unmarshal the JSON error:", err)
		http.Error(w, "Invalid JSON data actionForAgent", http.StatusBadRequest)
		return
	}

	message := fmt.Sprintf("Agent action received and saved successfully. Action is: %s and Version is: %s\n", agentAction.Deploy, agentAction.ClusterVersion)

	// Respond to the client
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(message))

	fmt.Printf("Running go routine to install or destroy the cluster. Action is: %s and Version is: %s\n", agentAction.Deploy, agentAction.ClusterVersion)
	go installOrDestroyCluster(agentAction.Deploy, agentAction.ClusterVersion)
}

func populateVersionToInstallerScript(clusterVersion string) {

	// Read the contents of the Terraform template file
	fmt.Println("Updating the installer script file")
	scriptContent, err := os.ReadFile("/app/cluster-installation-script.sh.template")
	if err != nil {
		fmt.Println("Cannot read install script file")
		return
	}

	//Create the Release channel from the cluster version provided from the user
	parts := strings.Split(clusterVersion, ".")
	var clusterReleaseChannnel string
	if len(parts) >= 2 {

		// Take the first two parts and concatenate "stable-" in front of them
		clusterReleaseChannnel = "stable-" + parts[0] + "." + parts[1]
	}

	// Replace the placeholder string with the generated public key path
	replacedClusterVersion := strings.ReplaceAll(string(scriptContent), "$CLUSTER_VERSION", clusterVersion)
	replacedChannel := strings.ReplaceAll(string(replacedClusterVersion), "$RELEASE_CHANNEL", clusterReleaseChannnel)
	err = os.WriteFile("/app/cluster-installation-script.sh", []byte(replacedChannel), 0644)
	if err != nil {
		fmt.Println("Cannot write the Installer script file")
		return
	}

}
