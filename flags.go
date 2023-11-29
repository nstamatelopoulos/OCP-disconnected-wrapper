package main

import (
	"fmt"
	"os"
	"regexp"
)

func consolidatedFlagCheckFunction(install bool, destroy bool, region string, clusterVersion string, init bool, helpflag bool, openshiftCNI bool) {
	singleFlagFunction(install, destroy, region, clusterVersion, init, helpflag, openshiftCNI)
	installFlagFunction(install, destroy, region, init)
	//clusterFlagFunction(install, destroy, region, clusterVersion, init)
	if install && (len(region) >= 0) {
		checkRegionString(regions, region)
	}
	if len(clusterVersion) > 0 {
		checkClusterVersionString(clusterVersion)
	}
}

func singleFlagFunction(install bool, destroy bool, region string, clusterVersion string, init bool, helpflag bool, cni bool) {

	if init && ((install || destroy || helpflag || cni) || (len(region) != 0 || (len(clusterVersion)) != 0)) {
		fmt.Println("Init flag cannot be used with any other flag but only alone. Please make sure no other flags are provided")
		os.Exit(1)
	} else if destroy && ((install || init || helpflag || cni) || (len(region) != 0 || (len(clusterVersion)) != 0)) {
		fmt.Println("Destroy flag cannot be used with any other flag but only alone. Please make sure no other flags are provided")
		os.Exit(1)
	} else if helpflag && ((install || init || destroy || cni) || (len(region) != 0 || (len(clusterVersion)) != 0)) {
		fmt.Println("Help flag cannot be used with any other flag but only alone. Please make sure no other flags are provided")
		os.Exit(1)
	}
}

func installFlagFunction(install bool, destroy bool, region string, init bool) {
	if install && (destroy || init) {
		fmt.Println("Install flag cannot be used with --init or --destroy. Please make sure these flags are not provided")
		os.Exit(1)
	} else if install && len(region) == 0 {
		fmt.Println("Please provide a region for the installation using --region flag")
		os.Exit(1)
	}
}

func checkRegionString(regions map[string]string, region string) {
	_, exists := regions[region]
	if !exists {
		fmt.Printf("The region: %s you provided is not a valid AWS region", region)
		os.Exit(1)
	}
}

func checkClusterVersionString(clusterVersion string) {
	regexPattern := `^4\.(1[0-7]|[0-9])\.(60|[0-5]?[0-9])$`
	matched, err := regexp.MatchString(regexPattern, clusterVersion)
	if err != nil {
		fmt.Println("Error in regular expression:", err)
		os.Exit(1)
	}
	if !matched {
		fmt.Printf("The provided cluster version: %s is not valid or out of the limits set in this program", clusterVersion)
		os.Exit(1)
	}
}

func flagsHelp() {
	fmt.Println("--init				This flag is to initialize the credentials of the program. It runs a prompt shell to add the required paths of the credentials")
	fmt.Println("--region			Set the AWS region")
	fmt.Println("--install			Install the chosen infrastructure")
	fmt.Println("--destroy			Destroy the chosen infrastructure")
	fmt.Println("--cluster			Enable cluster mode where a disconnected cluster will get installed along with the registry. Requires --cluster-version to be used")
	fmt.Println("--cluster-version		If --cluster flag is set use this flag to set the cluster version (e.g 4.12.13)")
	fmt.Println("--sdn              If --sdn flag is set the cluster will be installed with OpenShiftSDN CNI")
	fmt.Println("--help				Help")
}
