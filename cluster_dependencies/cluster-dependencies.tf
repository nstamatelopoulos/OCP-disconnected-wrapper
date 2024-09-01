resource "aws_route_table" "s3-route-table" {
  vpc_id     = var.Vpc_ID

  tags = {
    Name = "s3-route-table"
  }
}

resource "aws_route_table_association" "Private-1" {
  subnet_id      = aws_subnet.private-1.id
  route_table_id = aws_route_table.s3-route-table.id
}

resource "aws_route_table_association" "Private-2" {
  subnet_id      = aws_subnet.private-2.id
  route_table_id = aws_route_table.s3-route-table.id
}

resource "aws_route_table_association" "Private-3" {
  subnet_id      = aws_subnet.private-3.id
  route_table_id = aws_route_table.s3-route-table.id
}

resource "aws_security_group" "gateway-interfaces-sg" {
  name        = "allow_all"
  description = "Allow all inbound and outbound traffic"
  vpc_id      = var.Vpc_ID

  ingress {
    description      = "Allow from everywhere"
    from_port        = 0
    to_port          = 65535
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
  }

  egress {
    from_port        = 0
    to_port          = 65535
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
  }

  tags = {
    Name = "allow_all"
  }
}

resource "aws_vpc_endpoint" "s3" {
  vpc_id = var.Vpc_ID
  vpc_endpoint_type = "Gateway"
  service_name = "com.amazonaws.${var.Child_Region}.s3"
  route_table_ids = [aws_route_table.s3-route-table.id]

  tags = {
    Name = "s3.${var.Child_Region}.amazonaws.com"
  }
}

resource "aws_vpc_endpoint" "ec2" {
  vpc_id       = var.Vpc_ID
  service_name = "com.amazonaws.${var.Child_Region}.ec2"
  vpc_endpoint_type = "Interface"
  security_group_ids = [aws_security_group.gateway-interfaces-sg.id]
  subnet_ids = [aws_subnet.private-1.id, aws_subnet.private-2.id, aws_subnet.private-3.id]
  private_dns_enabled = true
  
  tags = {
    Name = "ec2.${var.Child_Region}.amazonaws.com"
  }
}

resource "aws_vpc_endpoint" "elb" {
  vpc_id       = var.Vpc_ID
  service_name = "com.amazonaws.${var.Child_Region}.elasticloadbalancing"
  vpc_endpoint_type = "Interface"
  security_group_ids = [aws_security_group.gateway-interfaces-sg.id]
  subnet_ids = [aws_subnet.private-1.id, aws_subnet.private-2.id, aws_subnet.private-3.id]
  private_dns_enabled = true

  tags = {
    Name = "elasticloadbalancing.${var.Child_Region}.amazonaws.com"
  }
}

variable Vpc_ID {
  type = string
}

resource "aws_subnet" "private-1" {
  vpc_id     = var.Vpc_ID
  cidr_block = "10.0.0.32/27"
  availability_zone = var.Child_Availability_Zone_A

  tags = {
    Name = "Private-1"
  }
}

resource "aws_subnet" "private-2" {
  vpc_id     = var.Vpc_ID
  cidr_block = "10.0.0.64/27"
  availability_zone = var.Child_Availability_Zone_B

  tags = {
    Name = "Private-2"
  }
}

resource "aws_subnet" "private-3" {
  vpc_id     = var.Vpc_ID
  cidr_block = "10.0.0.96/27"
  availability_zone = var.Child_Availability_Zone_C

  tags = {
    Name = "Private-3"
  }
}

# variable Child_Random_Suffix{
#   type = string
# }

variable Child_Availability_Zone_A {
  type = string
}

variable Child_Availability_Zone_B {
  type = string
}

variable Child_Availability_Zone_C {
  type = string
}

variable Child_Region {
  type = string
}

output Subnet_1 {
  value = aws_subnet.private-1.id
}

output Subnet_2 {
  value = aws_subnet.private-2.id
}

output Subnet_3 {
  value = aws_subnet.private-3.id
}