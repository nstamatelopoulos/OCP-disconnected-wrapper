package main

import (
	"fmt"
	"os"
	"regexp"
)

func consolidatedFlagCheckFunction(install bool, destroy bool, region string, clusterVersion string, init bool, helpflag bool, openshiftCNI bool, destroyCluster bool, addCluster bool, installConfig bool, force bool) {
	if addCluster {
		checkaddCluster(addCluster, clusterVersion, installConfig)
	} else if installConfig {
		checkInstallConfigFlag(installConfig, install, region, clusterVersion, addCluster, openshiftCNI)
	} else if force {
		checkForceFlag(destroy, force)
	} else {
		singleFlagFunction(install, destroy, region, clusterVersion, init, helpflag, openshiftCNI, destroyCluster)
		installFlagFunction(install, destroy, region, init)
	}
	//Check if the string flags have proper syntax and make sense.
	if install && (len(region) > 0) {
		checkRegionString(regions, region)
	}
	if len(clusterVersion) > 0 {
		checkClusterVersionString(clusterVersion)
	}
}

func singleFlagFunction(install bool, destroy bool, region string, clusterVersion string, init bool, helpflag bool, cni bool, destroyCluster bool) {

	if init && ((install || destroy || helpflag || cni || destroyCluster) || (len(region) != 0 || (len(clusterVersion)) != 0)) {
		fmt.Println("Init flag cannot be used with any other flag but only alone. Please make sure no other flags are provided")
		os.Exit(1)
	} else if destroy && ((install || init || helpflag || cni || destroyCluster) || (len(region) != 0 || (len(clusterVersion)) != 0)) {
		fmt.Println("Destroy flag cannot be used with any other flag but only alone. Please make sure no other flags are provided")
		os.Exit(1)
	} else if helpflag && ((install || init || destroy || cni || destroyCluster) || (len(region) != 0 || (len(clusterVersion)) != 0)) {
		fmt.Println("Help flag cannot be used with any other flag but only alone. Please make sure no other flags are provided")
		os.Exit(1)
	} else if destroyCluster && ((install || init || destroy || cni || helpflag) || (len(region) != 0 || (len(clusterVersion)) != 0)) {
		fmt.Println("Destroy-cluster flag cannot be used with any other flag but only alone. Please make sure no other flags are provided")
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

func checkaddCluster(addCluster bool, clusterVersion string, installConfig bool) {
	if !(addCluster && len(clusterVersion) > 0) && (installConfig || true) {
		fmt.Println("The --add-cluster flag need to be used only with -cluster-version one with a valid OCP version. Optional --custom-install-config flag")
		os.Exit(1)
	}
}

func checkInstallConfigFlag(installConfig bool, install bool, region string, clusterVersion string, addCluster bool, sdn bool) {
	// Install mode check
	if installConfig && install && len(clusterVersion) > 0 && len(region) > 0 && (sdn || !sdn) {
		return
	}
	// Add cluster mode check
	if installConfig && addCluster && len(clusterVersion) > 0 && (sdn || !sdn) {
		return
	}
	// If neither conditions were satisfied, print the error and exit
	fmt.Println("The --custom-install-config flag must be used with --install --region --cluster-version flags and as optional the --sdn flag")
	fmt.Println("OR if you add a cluster, it must be used with --add-cluster and --cluster-version flags and as optional the --sdn flag")
	os.Exit(1)
}

func checkForceFlag(destroy bool, force bool) {
	if !(force && destroy) {
		fmt.Println("The --force flag need to be used only with the --destroy flag in case the user knows what is doing.")
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
	fmt.Println("--init                     This flag is to initialize the credentials of the program. It runs a prompt shell to add the required paths of the credentials")
	fmt.Println("--region                   Set the AWS region")
	fmt.Println("--install                  Install the chosen infrastructure")
	fmt.Println("--destroy                  Destroy the chosen infrastructure")
	fmt.Println("--cluster-version          If --cluster flag is set use this flag to set the cluster version (e.g 4.12.13)")
	fmt.Println("--sdn                      If --sdn flag is set the cluster will be installed with OpenShiftSDN CNI. (Only for v4.14 installations and lower)")
	fmt.Println("--status                   Returns the status of the infrastructure provisioned. If Registry is healhty and if Cluster is installed or not. Agent must be healthy")
	fmt.Println("--add-cluster              Enables the user to install a cluster post deploying the mirror-registry. To be used with --cluster-version flag")
	fmt.Println("--destroy-cluster          Enables the user to destroy a cluster without destroying anything else. Mirror Registry is not affected only cluster is destroyed.")
	fmt.Println("--custom-install-config    Enables the user to use a custom install-config.yaml file. Requires a file with name 'install-config.yaml' under the OCPD cloned directory")
	fmt.Println("--help                     Help")
	fmt.Println("--version                  Prints OCPD release version")
}
