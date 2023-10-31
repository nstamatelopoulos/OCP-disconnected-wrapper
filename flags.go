package main

import (
	"fmt"
	"os"
	"regexp"
)

func consolidatedFlagCheckFunction(install bool, destroy bool, region string, cluster bool, clusterVersion string, init bool) {
	singleFlagFunction(install, destroy, region, cluster, clusterVersion, init)
	installFlagFunction(install, destroy, region, init)
	clusterFlagFunction(install, destroy, region, cluster, clusterVersion, init)
	checkRegionString(regions, region)
	checkClusterVersionString(clusterVersion)
}

func singleFlagFunction(install bool, destroy bool, region string, cluster bool, clusterVersion string, init bool) {

	if init && (install || destroy || cluster) || (len(region) != 0 || (len(clusterVersion)) != 0) {
		fmt.Println("Init flag cannot be used with any other flag but only alone. Please make sure no other flags are provided")
		os.Exit(1)
	} else if destroy && (install || init || cluster) || (len(region) != 0 || (len(clusterVersion)) != 0) {
		fmt.Println("Destroy flag cannot be used with any other flag but only alone. Please make sure no other flags are provided")
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

func clusterFlagFunction(install bool, destroy bool, region string, cluster bool, clusterVersion string, init bool) {
	if cluster && (!install || len(region) == 0) {
		fmt.Println("To use --cluster flag you need to also provide --install and --region flag with a valid AWS region (e.g eu-west-1)")
		os.Exit(1)
	} else if cluster && install && len(region) >= 0 && len(clusterVersion) == 0 {
		fmt.Println("When using --cluster flag you need to also use --cluster-version with a valid OCP version (e.g 4.13.11)")
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
	regexPattern := `^4\.[0-9]|1[0-5]\.[0-5]?[0-9]$`
	matched, err := regexp.MatchString(regexPattern, clusterVersion)
	if err != nil {
		fmt.Println("Error in regular expression:", err)
	}
	if !matched {
		fmt.Printf("The provided cluster version: %s is not valid or out of the limits set in this program", clusterVersion)
		os.Exit(1)
	}
}
