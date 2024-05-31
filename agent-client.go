package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

//==========================================================================================
// Agent/Client type of functions
//==========================================================================================

func MonitoringDeployment() {

	Ec2Url := GetInfraDetails("InstancePublicDNS")

	fmt.Println(Ec2Url)

	go ClientGetStatus(Ec2Url)

	select {}

}

type InfraState struct {
	RegistryHealth string
	ClusterStatus  string
}

func ClientGetStatus(url string) {

	fmt.Println("ClientGetStatus started")

	logFile, err := os.OpenFile("monitoring.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	defer logFile.Close()

	// Set log output to the file
	log.SetOutput(logFile)

	for {

		resp, err := http.Get(url + ":8090/status")
		if err != nil {
			log.Println("Error making GET request:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println("Error reading response body:", err)
			resp.Body.Close()
			time.Sleep(5 * time.Second)
		}

		resp.Body.Close()

		s := &InfraState{}

		if err := json.Unmarshal(body, s); err != nil {
			fmt.Printf("error Unmarshaling JSON: %v\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		fmt.Println("InfraState: ", s)

		time.Sleep(5 * time.Second)
	}

}
