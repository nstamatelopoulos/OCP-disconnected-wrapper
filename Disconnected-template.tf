provider "aws" {
  region = var.Region
}

data "aws_region" "current" {}
# test
resource "aws_vpc" "disconnected-vpc" {
  cidr_block       = "10.0.0.0/16"
  instance_tenancy = "default"
  enable_dns_support = true
  enable_dns_hostnames = true

  tags = {
    Name = "disconnected-vpc-${random_string.key_suffix.result}"
  }
}

resource "aws_subnet" "registry-subnet" {
  vpc_id     = aws_vpc.disconnected-vpc.id
  cidr_block = "10.0.0.0/28"
  availability_zone = var.Availability_Zone_A
  map_public_ip_on_launch = true

  tags = {
    Name = "registry-subnet"
  }
}

data "aws_route_table" "main-route-table" {
  vpc_id = aws_vpc.disconnected-vpc.id

  filter {
    name   = "association.main"
    values = ["true"]
  }
}

resource "aws_internet_gateway" "registry-gw" {
  vpc_id = aws_vpc.disconnected-vpc.id
  tags = {
    Name = "registry-gw"
  }
}

resource "aws_route" "registry-igw-route" {

  route_table_id            = data.aws_route_table.main-route-table.id
  destination_cidr_block    = "0.0.0.0/0"
  gateway_id                = aws_internet_gateway.registry-gw.id
}

resource "aws_security_group" "registry-sg" {
  name        = "allow_SSH_HTTPS"
  description = "allow_SSH_HTTPS"
  vpc_id      = aws_vpc.disconnected-vpc.id

  ingress {
    description      = "HTTPS from everywhere"
    from_port        = 8443
    to_port          = 8443
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
  }

  ingress {
    description      = "SSH from everywhere"
    from_port        = 22
    to_port          = 22
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
  }

  ingress {
    description      = "Agent port from everywhere"
    from_port        = 8090
    to_port          = 8090
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
  }

  egress {
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
  }
  

  tags = {
    Name = "allow_ssh_https"
  }
}

resource "random_string" "key_suffix" {
  length  = 6
  special = false
}

resource "aws_key_pair" "registry-key-pair" {
  key_name   = "registry-key-${random_string.key_suffix.result}"
  public_key = file(var.Public_Key_Path)
}

resource "aws_instance" "mirror-registry" {
  ami           = var.Ami_Id
  instance_type = "c5.xlarge"
  key_name      = aws_key_pair.registry-key-pair.key_name

  subnet_id     = aws_subnet.registry-subnet.id
  vpc_security_group_ids = [aws_security_group.registry-sg.id]

  user_data = templatefile("registry-mirror-script-terraform.tpl", {
        access_key_id     = aws_iam_access_key.Cluster_deployer_key.id
        access_key_secret = aws_iam_access_key.Cluster_deployer_key.secret
        random_token      = random_string.key_suffix.result
       })

root_block_device {
    volume_size = 700
    volume_type = "gp2"
  }

  tags = {
    Name = "registry-instance"
  }

}

locals {
  registry_public_dns = aws_instance.mirror-registry[*].public_dns
}

output "ec2_instance_public_dns" {
  description = "SSH command to connect to the EC2 instance"
  value       = aws_instance.mirror-registry.public_dns
}

output "ec2_instance_public_dns_for_user" {
  description = "SSH command to connect to the EC2 instance"
  value       = "To connect to the registry run ssh -i <your-private-key> ec2-user@${aws_instance.mirror-registry.public_dns}"
}

output "wait_for_initialization" {
  description = "Initialization instructions"
  value       = "The registry requires ~ 5 minutes to initialize. It will be ready when you see the READY file under /home/ec2-user/"
}

module Cluster_Dependencies {
  source = "./cluster_dependencies"
  count = var.Create_Cluster ? 1 : 0

  Vpc_ID = aws_vpc.disconnected-vpc.id
  Child_Availability_Zone_A = var.Availability_Zone_A
  Child_Availability_Zone_B = var.Availability_Zone_B
  Child_Availability_Zone_C = var.Availability_Zone_C
  Child_Region = var.Region
  #Child_Random_Suffix = random_string.key_suffix.result
}


output "region" {
  value = data.aws_region.current.name
  description = "The region of the infrastructure"
}
output "private_subnet_1_id" {
  value = var.Create_Cluster ? module.Cluster_Dependencies[0].Subnet_1 : "N/A"
  description = "The ID of the first private subnet"
}

output "private_subnet_2_id" {
  value = var.Create_Cluster ? module.Cluster_Dependencies[0].Subnet_2 : "N/A"
  description = "The ID of the first private subnet"
}

output "private_subnet_3_id" {
  value = var.Create_Cluster ? module.Cluster_Dependencies[0].Subnet_3 : "N/A"
  description = "The ID of the first private subnet"
}

output "ec2_private_hostname" {
  value = aws_instance.mirror-registry.private_dns
}

output "random_token" {
  value = random_string.key_suffix.result
}
