#!/bin/bash 

set -e

#==============================
echo "Starting Cluster Installation script"
#==============================
# Creating some variables
#==============================

homedir=/ec2-user

hostname=$(hostname)

CLUSTER_VERSION=$CLUSTER_VERSION

RELEASE_CHANNEL=$RELEASE_CHANNEL

export AWS_SHARED_CREDENTIALS_FILE=$homedir/.aws/credentials

#===============================================================
# Creating/Building imageset-config.yaml and install-config.yaml
#===============================================================

echo "Create the imageset-config.yaml file"

if ! [ -f $homedir/mirroring-workspace/imageset-config.yaml ]; then
cat <<EOF > "$homedir/mirroring-workspace/imageset-config.yaml"
apiVersion: mirror.openshift.io/v1alpha2
kind: ImageSetConfiguration
storageConfig:
  local:
    path: /home/ec2-user/mirroring-workspace/oc-mirror-metadata
mirror:
  platform:
    channels:
      - name: $RELEASE_CHANNEL
        minVersion: $CLUSTER_VERSION
        maxVersion: $CLUSTER_VERSION
EOF
else
  echo "The imageset-config.yaml file exists"
fi

#=================================
# Creating the install-config.yaml
#=================================

echo "Adding the CA to the install-config.yaml"

if ! grep -q 'additionalTrustBundle' $homedir/cluster/install-config.yaml; then
  echo "additionalTrustBundle:  |" >> $homedir/cluster/install-config.yaml
  echo "$(cat $homedir/registry-stuff/quay-rootCA/rootCA.pem)" | sed 's/^/      /' >> $homedir/cluster/install-config.yaml
else
  echo "The additionalTrustBundle already exists in the install-config"
fi

echo "Creating and adding the public-key to the install-config.yaml"

if ! [ -f $homedir/.ssh/cluster_key ]; then
  ssh-keygen -f $homedir/.ssh/cluster_key -t rsa -q -N ""
else
  echo "SSH key pair already exists"
fi

if ! grep -q 'sshKey' $homedir/cluster/install-config.yaml; then
  echo "sshKey: $(cat $homedir/.ssh/cluster_key.pub)" >> $homedir/cluster/install-config.yaml
else
  echo "SSH key already added to the install-config.yaml"
fi

echo "Adding pull-secret to the install-config.yaml"

if ! grep -q 'pullSecret' $homedir/cluster/install-config.yaml; then
  echo "pullSecret: '$(cat $homedir/.docker/config.json | jq -c )'" >> $homedir/cluster/install-config.yaml
else
  echo "The Pull Secret already exists in the install-config"
fi

echo "Cleanup the workspace"

rm -f $homedir/mirroring-workspace/oc-mirror.tar.gz
rm -f $homedir/pull-secret.template

#===========================================================================================================
# Downloading, unpacking installer, oc client and make changes to the manifest prior installing the cluster
#===========================================================================================================

echo "Starting Cluster deployment preparations"

echo "Creating the local binary /bin folder and exporting into PATH permanently"

if ! [ -d $homedir/bin ]; then
  mkdir $homedir/bin
  echo 'export PATH="/ec2-user/bin:$PATH"' >> $homedir/.bashrc
  source $homedir/.bashrc
  mv $homedir/mirroring-workspace/oc-mirror /ec2-user/bin
else
  echo "$homedir/bin directory already exists"
fi

echo "Copying the pull-secret to home folder of container user"

if ! [ -f /home/ec2-user/.docker ]; then
  mkdir /home/ec2-user/.docker
  cp $homedir/.docker/config.json /home/ec2-user/.docker/config.json
else
  echo "The pull-secret to home folder of container user already exists"
fi

echo "Exporting the CA trust to be used by the container"

export SSL_CERT_FILE=$homedir/registry-stuff/quay-rootCA/rootCA.pem

cd $homedir/mirroring-workspace

echo "Mirroring release images for version $CLUSTER_VERSION"
oc-mirror --config $homedir/mirroring-workspace/imageset-config.yaml docker://$hostname:8443 --verbose 1

cd $homedir/cluster

echo "Downloading openshift-installer for version $CLUSTER_VERSION"

if ! [ -f $homedir/bin/openshift-install ]; then
  wget -q https://mirror.openshift.com/pub/openshift-v4/clients/ocp/$CLUSTER_VERSION/openshift-install-linux.tar.gz

  echo "Unpacking openshift-installer"
  tar -xf openshift-install-linux.tar.gz
  rm -f openshift-install-linux.tar.gz
  mv ./openshift-install $homedir/bin
else
  echo "Openshift installer exists in the bin directory"
fi

echo "Downloading openshift-client"

if ! [ -f $homedir/bin/oc ]; then
  wget -q https://mirror.openshift.com/pub/openshift-v4/clients/ocp/$CLUSTER_VERSION/openshift-client-linux.tar.gz

  echo "Unpacking openshift-client"
  tar -xf openshift-client-linux.tar.gz
  rm -f openshift-client-linux.tar.gz
  mv ./oc $homedir/bin
else
  echo "The openshift client exists in the bin directory"
fi

echo "Creating manifests"

cp install-config.yaml install-config.yaml.bak

if ! [ -d $homedir/cluster/manifests ]; then
  openshift-install create manifests --dir ./

  echo "Changing the DNS cluster manifest to avoid ingress operator trying to add '*.apps' domain"

  cd $homedir/cluster/manifests
  sed '/baseDomain:/q' cluster-dns-02-config.yml > new-cluster-dns-02-config.yml && mv new-cluster-dns-02-config.yml cluster-dns-02-config.yml
else
  echo "There is already a manifest directory present"
fi

#============================
# Launch cluster installation
#============================

echo "Launch cluster installation"
cd $homedir/cluster
openshift-install create cluster --dir ./ --log-level=info &

#=============================================
# Adding manually *apps. domain using aws CLI
#=============================================

echo "Waiting for the LB and the cluster zone to be created so to apply the wildcard '*.apps.' record"

# I found that this is a way to check when the hosted zone is created but i know it is not the best way. I need to improve that in the future.
while ! grep -q "Waiting up to 40m0s" $homedir/cluster/.openshift_install.log; do
  echo "LB and zone are not ready yet"
  sleep 120
done

echo "LB and zone are ready. Adding manually *apps. domain using aws CLI"

region=$(jq -r '.aws.region' $homedir/cluster/metadata.json)
DOMAIN=$(jq -r '.aws.clusterDomain' $homedir/cluster/metadata.json)
HOSTED_ZONE_ID=$(aws route53 list-hosted-zones-by-name --dns-name $DOMAIN | jq -r '.HostedZones[0].Id')
Cluster_VPC_id=$(jq -r '.outputs.vpc_id.value' terraform.cluster.tfstate)

for ((i=0; i<=100; i++)); do
  LB_VPC_id=$(aws elb describe-load-balancers --region ${region} --load-balancer-names | jq -r ".LoadBalancerDescriptions[$i].VPCId")
  if [[ "$LB_VPC_id" == "$Cluster_VPC_id" ]]; then
    ELB_ALIAS_TARGET=$(aws elb describe-load-balancers --region ${region} --load-balancer-names | jq -r ".LoadBalancerDescriptions[$i].CanonicalHostedZoneNameID")
    ELB_DNS_NAME=$(aws elb describe-load-balancers --region ${region} --load-balancer-names | jq -r ".LoadBalancerDescriptions[$i].DNSName")
    echo "The domain of the zone is:" $DOMAIN
    echo "The elb hosted zone ID is: $ELB_ALIAS_TARGET" 
    echo "The elb DNS name is: "$ELB_D
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

#============================================================
# Allow in node SG Groups access from the registry using SSH
#============================================================

echo "Allow SSH access from registry to the cluster nodes"

# Master-nodes

SG_GROUP_NAME=$(jq -r '.cluster_id + "-master-sg"' terraform.tfvars.json)
SG_GROUP_ID=$(aws ec2 describe-security-groups --filters Name=vpc-id,Values=$Cluster_VPC_id --filters Name=tag:Name,Values=$SG_GROUP_NAME | jq -r '.SecurityGroups[0].GroupId')
aws ec2 authorize-security-group-ingress --group-id $SG_GROUP_ID --protocol tcp --port 22 --cidr 0.0.0.0/0

# Worker_nodes

SG_GROUP_NAME=$(jq -r '.cluster_id + "-worker-sg"' terraform.tfvars.json)
SG_GROUP_ID=$(aws ec2 describe-security-groups --filters Name=vpc-id,Values=$Cluster_VPC_id --filters Name=tag:Name,Values=$SG_GROUP_NAME | jq -r '.SecurityGroups[0].GroupId')
aws ec2 authorize-security-group-ingress --group-id $SG_GROUP_ID --protocol tcp --port 22 --cidr 0.0.0.0/0

#=================================
# Disable default Catalog Sources
#=================================

export KUBECONFIG=$homedir/cluster/auth/kubeconfig

oc patch OperatorHub cluster --type json \
    -p '[{"op": "add", "path": "/spec/disableAllDefaultSources", "value": true}]'