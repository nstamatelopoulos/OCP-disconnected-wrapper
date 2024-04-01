package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	url = "https://localhost:8443"
)

var (
	isRegistryHealthy bool
	healthMutex       sync.Mutex
)

func main() {

	go monitorRegistry(url)

	agentHTTPServer()

	select {}
}

// func hello(w http.ResponseWriter, req *http.Request) {

// 	fmt.Fprintf(w, "hello\n")
// }

// func headers(w http.ResponseWriter, req *http.Request) {

// 	for name, headers := range req.Header {
// 		for _, h := range headers {
// 			fmt.Fprintf(w, "%v: %v\n", name, h)
// 		}
// 	}
// }

func agentHTTPServer() {

	http.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		if getHealthStatus() {
			fmt.Fprintf(w, "Registry is healthy!\n")
		} else {
			fmt.Fprintf(w, "Registry is not healthy!\n")
		}
	})

	//http.HandleFunc("/headers", headers)

	fmt.Println("Starting HTTP Agent")
	if err := http.ListenAndServe(":8090", nil); err != nil {
		fmt.Printf("Error Starting HTTP Agent: %s\n", err)
	}

}

// Monitors the Registry by testing port 8443 every 10 seconds
func monitorRegistry(url string) {

	for {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
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
		time.Sleep(10 * time.Second)
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
