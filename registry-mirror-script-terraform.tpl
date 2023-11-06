#!/bin/bash 

#=================================================================================================
echo "Creating root directory"
#=================================================================================================
# Creating some variables
#=================================================================================================

homedir=/home/ec2-user

hostname=$(hostname)

CREATE_CLUSTER=true

CLUSTER_VERSION=4.14.1

RELEASE_CHANNEL=stable-4.14

RANDOM_VALUE=$RANDOM

export AWS_SHARED_CREDENTIALS_FILE=$homedir/.aws/credentials

Cluster_VPC_id=${cluster_VPC_id}

#=================================================================================================
# Creating/Installing some dependencies
#=================================================================================================

mkdir $homedir/registry-stuff && cd $homedir/registry-stuff

echo "Installing podman and wget"

sudo yum install wget -y -q
sudo yum install podman -y -q
sudo yum install jq -y -q
sudo yum install unzip -y -q

curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" 
unzip awscliv2.zip
sudo ./aws/install

#=================================================================================================
# Downloading and setting up mirror-registry
#=================================================================================================

echo "Getting mirror-registry package"

wget -q https://developers.redhat.com/content-gateway/rest/mirror/pub/openshift-v4/clients/mirror-registry/latest/mirror-registry.tar.gz

echo "Unpacking the tar file"

tar -xf mirror-registry.tar.gz

echo "Installing mirror-registry"

sudo ./mirror-registry install --quayHostname=$(hostname) --quayRoot=$homedir/registry-stuff > registry-install.log

echo "Get the password for the registry"

password=$(cat registry-install.log | grep -o 'credentials (init, [^)]*)' | sed "s/credentials (init, \([^)]*\))/\1/")

password_hash=$(echo -n "init:$${password}" | base64 -w0)

#=================================================================================================
# Creating the global pull-secret for the ec2-user using the init password and hostname
#=================================================================================================

echo "Create the pull-secret with the credentials and store it under ./docker/config.json"

mkdir $homedir/.docker

cat <<EOF > "$homedir/pull-secret.template"
{
  "auths": {
    "REGISTRY-HOSTNAME:8443": {
      "auth": "CREDENTIALS",
      "email": "registry@example.com"
    },
    "cloud.openshift.com": {
      "auth": "b3BlbnNoaWZ0LXJlbGVhc2UtZGV2K29jbV9hY2Nlc3NfY2M4NjlhYzRiNDk1NDEwZmEwZjRhNzY1Y2RlMzQzMDk6QzI0VjdGNEtaTk9UTzFEU0RYVFROSVFSMVJHR1o2M1lZNFVZV01QQTNIOVI4SlFDNkg4RVFXQkpEMVRBMkJDSw==",
      "email": "nstamate@redhat.com"
    },
    "quay.io": {
      "auth": "b3BlbnNoaWZ0LXJlbGVhc2UtZGV2K29jbV9hY2Nlc3NfY2M4NjlhYzRiNDk1NDEwZmEwZjRhNzY1Y2RlMzQzMDk6QzI0VjdGNEtaTk9UTzFEU0RYVFROSVFSMVJHR1o2M1lZNFVZV01QQTNIOVI4SlFDNkg4RVFXQkpEMVRBMkJDSw==",
      "email": "nstamate@redhat.com"
    },
    "registry.connect.redhat.com": {
      "auth": "fHVoYy1wb29sLTg4OTNlM2RjLWY4YzItNDA2Zi05OWE2LWRiZDc2MzdjN2NiNTpleUpoYkdjaU9pSlNVelV4TWlKOS5leUp6ZFdJaU9pSTBNVFUxTm1KbE0yTTRPR1kwWm1NeVltSm1OMlk0WTJJeFpqYzRZMk00WWlKOS5DTEdNLU4zZVNUcWp3UGk4Si10Wm1XaUs4YmdER2tUbVRreWJzM1FMdjlLeXJON29POXc4ektTeEZYYkkzeFNFTTJwal9XWWI4d29iOUdDX0FrLUN6YUdPTWdkbkJFbExJMXlSSVJKakVXeWl2U0lqNG5EZW40YmhvcndhTXc3c09FbWQydXVDclJhRzJRbFg4NkdFOTFLNnVMUjdqTG9hNkNXVUxmV2FoTzJnUDhXM3dFeTR4YjFiYVN0R09uQ3A0bTNhNzJSWEtPSEtjVm16UlB3SlhOTnVxZk9xUS1RbzVJSW9IQUx0elVoTkRfa1cxUkFtS1Q5bjRId3R3ZDlDRDRabjBZejByNW5fampHNWd5bTRncEplRUk2bndVb0Qzbk9nREE2bUFfR0NJTVNUVUlQZ01qeW9QZnBzUmRuZkNSN180ai1GQmlBaEdGZzQwRjhWUk9TSWV4blRYYUNyX0Z1NWo3R19QVXJkS0UyWElEMFdROEZ2Y1hhdmlsS2pvRTdjQVlfNzhQckhBRVAtVElCcGNwbzlBOE9ueXFwM0hYMy1QUlRTMFlLQ0tTdzZqQktzYzVCMmE5eHFJV1ZESGhxTllVMUt3YzZDcFE0UGw5T3lnOUJiWGZqS0hMM3VPMHRHTlktMjBGcHh0Wmhzd0FrY2JkVlM4ekQ5aXhtVzFvVXhUNlkzaUVaVlM4QWg4OFpJaXJUVnhDY3JoS0tJa3QzY0NabTlNLUZTbG91bXZtSVp5Y0ktcEpKZzFIQnBGdUc0NG5wVm9VT2VZUTJ3RElSX3loOS1IVzByMF82LUdjaFJ0Wi1TLW1DeEFrLXphYmFSR1drb1dvNGpoRTBLSllSaWRnTU9idDM3dWV3N3J5bXl2X1JmeTlOMmFmS1pUTk1BcjBvS2dJbw==",
      "email": "nstamate@redhat.com"
    },
    "registry.redhat.io": {
      "auth": "fHVoYy1wb29sLTg4OTNlM2RjLWY4YzItNDA2Zi05OWE2LWRiZDc2MzdjN2NiNTpleUpoYkdjaU9pSlNVelV4TWlKOS5leUp6ZFdJaU9pSTBNVFUxTm1KbE0yTTRPR1kwWm1NeVltSm1OMlk0WTJJeFpqYzRZMk00WWlKOS5DTEdNLU4zZVNUcWp3UGk4Si10Wm1XaUs4YmdER2tUbVRreWJzM1FMdjlLeXJON29POXc4ektTeEZYYkkzeFNFTTJwal9XWWI4d29iOUdDX0FrLUN6YUdPTWdkbkJFbExJMXlSSVJKakVXeWl2U0lqNG5EZW40YmhvcndhTXc3c09FbWQydXVDclJhRzJRbFg4NkdFOTFLNnVMUjdqTG9hNkNXVUxmV2FoTzJnUDhXM3dFeTR4YjFiYVN0R09uQ3A0bTNhNzJSWEtPSEtjVm16UlB3SlhOTnVxZk9xUS1RbzVJSW9IQUx0elVoTkRfa1cxUkFtS1Q5bjRId3R3ZDlDRDRabjBZejByNW5fampHNWd5bTRncEplRUk2bndVb0Qzbk9nREE2bUFfR0NJTVNUVUlQZ01qeW9QZnBzUmRuZkNSN180ai1GQmlBaEdGZzQwRjhWUk9TSWV4blRYYUNyX0Z1NWo3R19QVXJkS0UyWElEMFdROEZ2Y1hhdmlsS2pvRTdjQVlfNzhQckhBRVAtVElCcGNwbzlBOE9ueXFwM0hYMy1QUlRTMFlLQ0tTdzZqQktzYzVCMmE5eHFJV1ZESGhxTllVMUt3YzZDcFE0UGw5T3lnOUJiWGZqS0hMM3VPMHRHTlktMjBGcHh0Wmhzd0FrY2JkVlM4ekQ5aXhtVzFvVXhUNlkzaUVaVlM4QWg4OFpJaXJUVnhDY3JoS0tJa3QzY0NabTlNLUZTbG91bXZtSVp5Y0ktcEpKZzFIQnBGdUc0NG5wVm9VT2VZUTJ3RElSX3loOS1IVzByMF82LUdjaFJ0Wi1TLW1DeEFrLXphYmFSR1drb1dvNGpoRTBLSllSaWRnTU9idDM3dWV3N3J5bXl2X1JmeTlOMmFmS1pUTk1BcjBvS2dJbw==",
      "email": "nstamate@redhat.com"
    }
  }
}
EOF

cp $homedir/pull-secret.template $homedir/.docker/config.json

sed -i "s/CREDENTIALS/$password_hash/g" $homedir/.docker/config.json
sed -i "s/REGISTRY-HOSTNAME/$hostname/g" $homedir/.docker/config.json

#=================================================================================================
# Creating the filesystem required, Downloading oc-mirror and fixing the CA trust.
#=================================================================================================

echo "Create oc-mirror and cluster workspace directory"

mkdir $homedir/mirroring-workspace
mkdir $homedir/cluster
mkdir $homedir/.aws

echo "Downloading oc-mirror"

wget -q https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/stable/oc-mirror.tar.gz -P $homedir/mirroring-workspace/

tar -xf $homedir/mirroring-workspace/oc-mirror.tar.gz -C $homedir/mirroring-workspace/

chmod +x $homedir/mirroring-workspace/oc-mirror

echo "Adding the rootCA to the host so registry to be trusted"

sudo cp $homedir/registry-stuff/quay-rootCA/rootCA.pem /etc/pki/ca-trust/source/anchors/

sudo update-ca-trust

echo "Create the imageset-config.yaml and install-config.yaml template file"

#=================================================================================================
# Creating/Building imageset-config.yaml and install-config.yaml
#=================================================================================================

cat <<EOF > "$homedir/mirroring-workspace/imageset-config.yaml"
apiVersion: mirror.openshift.io/v1alpha2
kind: ImageSetConfiguration
storageConfig:
  local:
    path: $homedir/oc-mirror-metadata
mirror:
  platform:
    channels:
      - name: $RELEASE_CHANNEL
        minVersion: $CLUSTER_VERSION
        maxVersion: $CLUSTER_VERSION
EOF

cat <<EOF > "$homedir/cluster/install-config.yaml"
apiVersion: v1
baseDomain: emea.aws.cee.support
credentialsMode: Passthrough
compute:
- architecture: amd64
  hyperthreading: Enabled
  name: worker
  platform: {}
  replicas: 3
controlPlane:
  architecture: amd64
  hyperthreading: Enabled
  name: master
  platform: {}
  replicas: 3
metadata:
  creationTimestamp: null
  name: disconnected-$RANDOM_VALUE
networking:
  clusterNetwork:
  - cidr: 10.128.0.0/14
    hostPrefix: 23
  machineNetwork:
  - cidr: 10.0.0.32/27
  - cidr: 10.0.0.64/27
  - cidr: 10.0.0.96/27
  networkType: OVNKubernetes
  serviceNetwork:
  - 172.30.0.0/16
platform:
  aws:
    region: ${region}
    subnets:
    - ${private_subnet_1}
    - ${private_subnet_2}
    - ${private_subnet_3}
publish: Internal
imageContentSources:
  - mirrors:
    - $hostname:8443/openshift/release
    source: quay.io/openshift-release-dev/ocp-v4.0-art-dev
  - mirrors:
    - $hostname:8443/openshift/release-images
    source: quay.io/openshift-release-dev/ocp-release
EOF

echo "Fixing directory permissions"

chown -R ec2-user:ec2-user $homedir/mirroring-workspace/
chown -R ec2-user:ec2-user $homedir/cluster

#=================================================================================================
# Creating the .aws/Credentials file with the static credentials from the cluster deployer user
#=================================================================================================

mkdir $homedir/.aws

cat <<EOF > $homedir/.aws/credentials
[default]
aws_access_key_id = ${access_key_id}
aws_secret_access_key = ${access_key_secret}
EOF

chown -R ec2-user:ec2-user $homedir/.aws/

#=================================================================================================
# Creating the install-config.yaml
#=================================================================================================

echo "Adding the CA to the install-config.yaml"

echo "additionalTrustBundle:  |" >> $homedir/cluster/install-config.yaml
echo "$(cat $homedir/registry-stuff/quay-rootCA/rootCA.pem)" | sed 's/^/      /' >> $homedir/cluster/install-config.yaml

echo "Creating and adding the public-key to the install-config.yaml"

runuser -u ec2-user -- ssh-keygen -f $homedir/.ssh/cluster_key -t rsa -q -N ""

echo "sshKey: $(cat $homedir/.ssh/cluster_key.pub)" >> $homedir/cluster/install-config.yaml

echo "Adding pull-secret to the install-config.yaml"

echo "pullSecret: '$(cat $homedir/.docker/config.json | jq -c )'" >> $homedir/cluster/install-config.yaml

echo "Cleanup the workspace"

rm $homedir/mirroring-workspace/oc-mirror.tar.gz
rm $homedir/pull-secret.template

#=================================================================================================
# Let the user know that the mirror registry is ready to use
#=================================================================================================

echo "Registry is ready to mirror"

cat <<EOF > "$homedir/READY"
The registry was initialized successfully!
EOF

#=================================================================================================
# If the user selected to create a cluster along with the Registry do the appropriate actions.
#=================================================================================================

if [ "$CREATE_CLUSTER" = "true" ]
then
#===========================================================================================================
# Downloading, unpacking installer, oc client and make changes to the manifest prior installing the cluster
#===========================================================================================================

   echo "Starting Cluster deployment preparations"

   echo "Mirroring release images for version $CLUSTER_VERSION"
   cd $homedir/mirroring-workspace/
   runuser -u ec2-user -- ./oc-mirror --config imageset-config.yaml docker://$hostname:8443

   cd $homedir/cluster
   echo "Downloading openshift-installer for version $CLUSTER_VERSION"
   wget https://mirror.openshift.com/pub/openshift-v4/clients/ocp/$CLUSTER_VERSION/openshift-install-linux.tar.gz

   echo "Unpacking openshift-installer"
   tar -xf openshift-install-linux.tar.gz
   rm openshift-install-linux.tar.gz
   mv ./openshift-install /usr/local/bin

   echo "Downloading openshift-client"
   wget https://mirror.openshift.com/pub/openshift-v4/clients/ocp/$CLUSTER_VERSION/openshift-client-linux.tar.gz

   echo "Unpacking openshift-client"
   tar -xf openshift-client-linux.tar.gz
   rm openshift-client-linux.tar.gz
   mv ./oc /usr/local/bin

   echo "Creating manifests"
   cp install-config.yaml install-config.yaml.bak
   runuser -u ec2-user -- openshift-install create manifests --dir ./

   echo "Changing the DNS cluster manifest to avoid ingress operator try to add "*.apps" domain"
   cd $homedir/cluster/manifests
   sed '/baseDomain:/q' cluster-dns-02-config.yml > new-cluster-dns-02-config.yml && mv new-cluster-dns-02-config.yml cluster-dns-02-config.yml
   chown -R ec2-user:ec2-user $homedir/cluster

#===========================================================================================================
# Launch cluster installation
#===========================================================================================================
   echo "Launch cluster installation"
   cd $homedir/cluster
   runuser -u ec2-user -- openshift-install create cluster --dir ./ --log-level=info &

#===========================================================================================================
# Adding manually *apps. domain using aws CLI
#===========================================================================================================

  echo "Waiting for the LB and the cluster zone to be created so to apply the wildcard "apps." record"

# I found that this is a way to check when the hosted zone is created but i know it is not the best way. I need to improve that in the future.
  while ! grep -q "Waiting up to 40m0s" $homedir/cluster/.openshift_install.log ; do

        echo "LB and zone are not yet ready yet"
        sleep 120

  done

    echo "LB and zone are ready. Adding manually *apps. domain using aws CLI"

    DOMAIN=disconnected-$RANDOM_VALUE.emea.aws.cee.support
    HOSTED_ZONE_ID=$(aws route53 list-hosted-zones-by-name --dns-name $DOMAIN | jq -r '.HostedZones[0].Id')
    for ((i=0; i<=100; i++)); do
        VPC_id=$(aws elb describe-load-balancers --region ${region} --load-balancer-names | jq -r ".LoadBalancerDescriptions[$i].VPCId")
        if [[ "$VPC_id" == "$Cluster_VPC_id" ]]; then
        ELB_ALIAS_TARGET=$(aws elb describe-load-balancers --region ${region}  --load-balancer-names | jq -r ".LoadBalancerDescriptions[$i].CanonicalHostedZoneNameID")
        ELB_DNS_NAME=$(aws elb describe-load-balancers --region ${region}  --load-balancer-names | jq -r ".LoadBalancerDescriptions[$i].DNSName")
        echo "The domain of the zone is:" $DOMAIN
        echo "The elb hosted zone ID is: $ELB_ALIAS_TARGET" 
        echo "The elb DNS name is: "$ELB_DNS_NAME
        break
        else 
        echo "No LB found in VPC with ID $Cluster_VPC_id"
        fi
    done
    
    aws route53 change-resource-record-sets \
    --hosted-zone-id "$HOSTED_ZONE_ID" \
    --change-batch '{
      "Changes": [
        {
          "Action": "CREATE",
          "ResourceRecordSet": {
            "Name": "*.apps.'$DOMAIN'",
            "Type": "A",
            "AliasTarget": {
              "HostedZoneId": "'$ELB_ALIAS_TARGET'",
              "DNSName": "'$ELB_DNS_NAME'",
              "EvaluateTargetHealth": false
            }
          }
        }
      ]
    }'

fi

#===========================================================================================================
# Allow in node SG Groups access from the registry using SSH
#===========================================================================================================

echo "Allow SSH access from registry to the cluster nodes"

VPC_ID=$(jq -r '.outputs.vpc_id.value' terraform.cluster.tfstate)

# Master-nodes

SG_GROUP_NAME=$(jq -r '.cluster_id + "-master-sg"' terraform.tfvars.json)
SG_GROUP_ID=$(aws ec2 describe-security-groups --filters Name=vpc-id,Values=$VPC_ID --filters Name=tag:Name,Values=$SG_GROUP_NAME | jq -r '.SecurityGroups[0].GroupId')
aws ec2 authorize-security-group-ingress --group-id $SG_GROUP_ID --protocol tcp --port 22 --cidr 0.0.0.0/0

# Worker_nodes

SG_GROUP_NAME=$(jq -r '.cluster_id + "-worker-sg"' terraform.tfvars.json)
SG_GROUP_ID=$(aws ec2 describe-security-groups --filters Name=vpc-id,Values=$VPC_ID --filters Name=tag:Name,Values=$SG_GROUP_NAME | jq -r '.SecurityGroups[0].GroupId')
aws ec2 authorize-security-group-ingress --group-id $SG_GROUP_ID --protocol tcp --port 22 --cidr 0.0.0.0/0
