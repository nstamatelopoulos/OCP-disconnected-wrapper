provider "aws" {
  region = "eu-west-1"
}

resource "aws_vpc" "disconnected-vpc" {
  cidr_block       = "10.0.0.0/16"
  instance_tenancy = "default"
  enable_dns_support = true
  enable_dns_hostnames = true

  tags = {
    Name = "disconnected-vpc"
  }
}

resource "aws_subnet" "registry-subnet" {
  vpc_id     = aws_vpc.disconnected-vpc.id
  cidr_block = "10.0.0.0/28"
  availability_zone = "eu-west-1a"
  map_public_ip_on_launch = true

  tags = {
    Name = "registry-subnet"
  }
}

data "aws_route_table" "route-table" {
 # subnet_id = "subnet-000eda3c528a3ec5f"
 vpc_id = aws_vpc.disconnected-vpc.id
}

resource "aws_internet_gateway" "registry-gw" {
  vpc_id = aws_vpc.disconnected-vpc.id
  tags = {
    Name = "registry-gw"
  }
}

resource "aws_route" "registry-igw-route" {
  route_table_id            = data.aws_route_table.route-table.id
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

resource "aws_instance" "mirror-registry" {
  ami           = "ami-0f0f1c02e5e4d9d9f"
  instance_type = "c5.xlarge"
  key_name      = "nstamate"

  subnet_id     = aws_subnet.registry-subnet.id
  vpc_security_group_ids = [aws_security_group.registry-sg.id]

  user_data = <<-EOF
    #!/bin/bash
    echo "export EC2_PUBLIC_DNS=\$(curl -s http://169.254.169.254/latest/meta-data/public-hostname)" >> /etc/profile
    EOF

  provisioner "file" {
    source      = "./pull-secret.template"
    destination = "/home/ec2-user/pull-secret.template"

    connection {
      type        = "ssh"
      host        = self.public_ip
      user        = "ec2-user"
      private_key = file("/home/nstamate/nstamate.pem")
    }
  }

  provisioner "remote-exec" {
    script = "./registry-mirror-script-terraform.sh"
    connection {
      type        = "ssh"
      host        = self.public_ip
      user        = "ec2-user"
      private_key = file("/home/nstamate/nstamate.pem")
    }
  }


  root_block_device {
    volume_size = 700
    volume_type = "gp2"
  }

  tags = {
    Name = "registry-instance"
  }
}
