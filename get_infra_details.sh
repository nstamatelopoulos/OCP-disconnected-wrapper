#!/bin/bash

# Extract the values from terraform outputs
region=$(terraform output -raw region)
instance_dns=$(terraform output -raw ec2_instance_public_dns)
subnet1_id=$(terraform output -raw private_subnet_1_id)
subnet2_id=$(terraform output -raw private_subnet_2_id)
subnet3_id=$(terraform output -raw private_subnet_3_id)

# Create a JSON object and write it to infra_details.json
cat <<EOF > infra_details.json
{
  "aws_region":"$region"
  "instance_public_dns": "$instance_dns",
  "private_subnet_1": "$subnet1_id",
  "private_subnet_2": "$subnet2_id",
  "private_subnet_3": "$subnet3_id"
}
EOF

