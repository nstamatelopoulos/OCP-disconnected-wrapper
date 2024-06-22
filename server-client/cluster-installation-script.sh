#!/bin/bash 

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

#=================================
# Creating the install-config.yaml
#=================================

echo "Adding the CA to the install-config.yaml"

echo "additionalTrustBundle:  |" >> $homedir/cluster/install-config.yaml
echo "$(cat $homedir/registry-stuff/quay-rootCA/rootCA.pem)" | sed 's/^/      /' >> $homedir/cluster/install-config.yaml

echo "Creating and adding the public-key to the install-config.yaml"

ssh-keygen -f $homedir/.ssh/cluster_key -t rsa -q -N ""

echo "sshKey: $(cat $homedir/.ssh/cluster_key.pub)" >> $homedir/cluster/install-config.yaml

echo "Adding pull-secret to the install-config.yaml"

echo "pullSecret: '$(cat $homedir/.docker/config.json | jq -c )'" >> $homedir/cluster/install-config.yaml

echo "Cleanup the workspace"

rm -f $homedir/mirroring-workspace/oc-mirror.tar.gz
rm -f $homedir/pull-secret.template

#===========================================================================================================
# Downloading, unpacking installer, oc client and make changes to the manifest prior installing the cluster
#===========================================================================================================

   echo "Starting Cluster deployment preparations"

   echo "Creating the local binary /bin folder and exporting into PATH permanently"

   mkdir $homedir/bin
   echo 'export PATH="/ec2-user/bin:$PATH"' >> $homedir/.bashrc
   source $homedir/.bashrc

   echo "Mirroring release images for version $CLUSTER_VERSION"
   oc-mirror --config $homedir/mirroring-workspace/imageset-config.yaml docker://$hostname:8443

   cd $homedir/cluster
   echo "Downloading openshift-installer for version $CLUSTER_VERSION"
   wget -q https://mirror.openshift.com/pub/openshift-v4/clients/ocp/$CLUSTER_VERSION/openshift-install-linux.tar.gz

   echo "Unpacking openshift-installer"
   tar -xf openshift-install-linux.tar.gz
   rm -f openshift-install-linux.tar.gz
   mv ./openshift-install $homedir/bin

   echo "Downloading openshift-client"
   wget -q https://mirror.openshift.com/pub/openshift-v4/clients/ocp/$CLUSTER_VERSION/openshift-client-linux.tar.gz

   echo "Unpacking openshift-client"
   tar -xf openshift-client-linux.tar.gz
   rm -f openshift-client-linux.tar.gz
   mv ./oc $homedir/bin

   echo "Creating manifests"
   cp install-config.yaml install-config.yaml.bak
   openshift-install create manifests --dir ./

   echo "Changing the DNS cluster manifest to avoid ingress operator try to add "*.apps" domain"
   cd $homedir/cluster/manifests
   sed '/baseDomain:/q' cluster-dns-02-config.yml > new-cluster-dns-02-config.yml && mv new-cluster-dns-02-config.yml cluster-dns-02-config.yml
   chown -R ec2-user:ec2-user $homedir/cluster

#============================
# Launch cluster installation
#============================

   echo "Launch cluster installation"
   cd $homedir/cluster
   openshift-install create cluster --dir ./ --log-level=info &

#=============================================
# Adding manually *apps. domain using aws CLI
#=============================================

  echo "Waiting for the LB and the cluster zone to be created so to apply the wildcard "apps." record"

# I found that this is a way to check when the hosted zone is created but i know it is not the best way. I need to improve that in the future.
  while ! grep -q "Waiting up to 40m0s" $homedir/cluster/.openshift_install.log ; do

        echo "LB and zone are not ready yet"
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

#============================================================
# Allow in node SG Groups access from the registry using SSH
#============================================================

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

fi

#=================================
# Disable default Catalog Sources
#=================================

export KUBECONFIG=$homedir/cluster/auth/kubeconfig

oc patch OperatorHub cluster --type json \
    -p '[{"op": "add", "path": "/spec/disableAllDefaultSources", "value": true}]'