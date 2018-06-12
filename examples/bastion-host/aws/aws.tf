provider "aws" {
  version = "~> 1.22.0"

  region = "${var.aws_region}"
}

# KEYS
resource "aws_key_pair" "host_key_pair" {
  key_name   = "${var.host_key_name}"
  public_key = "${file(pathexpand("~/.ssh/tf-examples/${var.host_key_name}.pub"))}"
}

# VPC
resource "aws_vpc" "default" {
  cidr_block = "12.0.0.0/16"
}

resource "aws_internet_gateway" "default" {
  vpc_id = "${aws_vpc.default.id}"
}

resource "aws_route" "internet_access" {
  route_table_id         = "${aws_vpc.default.main_route_table_id}"
  destination_cidr_block = "0.0.0.0/0"
  gateway_id             = "${aws_internet_gateway.default.id}"
}

resource "aws_subnet" "default" {
  vpc_id                  = "${aws_vpc.default.id}"
  cidr_block              = "12.0.1.0/24"
  map_public_ip_on_launch = true
}

# Secgroups
## Bastion instances
resource "aws_security_group" "bastion_host" {
  name        = "bastion-host"
  description = "Security Group for bastion host"
  vpc_id      = "${aws_vpc.default.id}"

  tags {
    Terraform = "true"
  }
}

resource "aws_security_group_rule" "bastion_host_egress_ssh" {
  type              = "egress"
  security_group_id = "${aws_security_group.bastion_host.id}"

  from_port = 22
  to_port   = 22
  protocol  = "tcp"

  source_security_group_id = "${aws_security_group.bastion_host_access.id}"
}

resource "aws_security_group_rule" "bastion_host_ingress_allow_ssh" {
  type              = "ingress"
  security_group_id = "${aws_security_group.bastion_host.id}"

  from_port        = 22
  to_port          = 22
  protocol         = "tcp"
  cidr_blocks      = ["0.0.0.0/0"]
  ipv6_cidr_blocks = ["::/0"]
}

resource "aws_security_group_rule" "bastion_host_outbound_internet_access" {
  description       = "Outbound internet access"
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = "${aws_security_group.bastion_host.id}"
}

# This sec group will later be attached to all instance which should be accessible
# via the bastion host!
resource "aws_security_group" "bastion_host_access" {
  name        = "inbound-bastion-host-access"
  description = "Security group for allowing inbound ssh from the bastion-host"
  vpc_id      = "${aws_vpc.default.id}"

  tags {
    Terraform = "true"
  }
}

resource "aws_security_group_rule" "bastion_host_access_inbound_ssh" {
  type = "ingress"

  security_group_id = "${aws_security_group.bastion_host_access.id}"

  from_port = 22
  to_port   = 22
  protocol  = "tcp"

  source_security_group_id = "${aws_security_group.bastion_host.id}"
}

## Swarm instance
resource "aws_security_group" "swarm_instances" {
  name        = "swarm-instances"
  description = "Security group for the swarm instances"
  vpc_id      = "${aws_vpc.default.id}"

  tags {
    Terraform = "true"
  }
}

resource "aws_security_group_rule" "outbound_internet_access" {
  description       = "Outbound internet access"
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = "${aws_security_group.swarm_instances.id}"
}

# Instances
## Bastion host
resource "aws_instance" "bastion_host" {
  tags {
    Name      = "bastion-host"
    Terraform = "true"
  }

  connection {
    type        = "ssh"
    user        = "${var.user}"
    private_key = "${file(pathexpand("~/.ssh/tf-examples/${var.host_key_name}"))}"
  }

  instance_type               = "${var.instance_type}"
  ami                         = "${var.ami}"
  key_name                    = "${var.host_key_name}"
  vpc_security_group_ids      = ["${aws_security_group.bastion_host.id}"]
  subnet_id                   = "${aws_subnet.default.id}"
  associate_public_ip_address = "true"

  provisioner "remote-exec" {
    inline = [
      "sudo hostname bastion-host",
    ]
  }
}

## Docker Swarm 
resource "aws_instance" "swarm_manager" {
  tags {
    Name      = "swarm-manager"
    Terraform = "true"
  }

  connection {
    type        = "ssh"
    host        = "${self.private_ip}"
    user        = "${var.user}"
    private_key = "${file(pathexpand("~/.ssh/tf-examples/${var.host_key_name}"))}"

    bastion_host        = "${aws_instance.bastion_host.public_ip}"
    bastion_user        = "${var.user}"
    bastion_private_key = "${file(pathexpand("~/.ssh/tf-examples/${var.host_key_name}"))}"
  }

  instance_type = "${var.instance_type}"
  ami           = "${var.ami}"
  key_name      = "${var.host_key_name}"

  vpc_security_group_ids = [
    "${aws_security_group.swarm_instances.id}",
    "${aws_security_group.bastion_host_access.id}",
  ]

  subnet_id = "${aws_subnet.default.id}"

  # NEEDED to install Docker via 'remote-exec'. Consider a pre-backed AMI here
  associate_public_ip_address = "true"

  provisioner "remote-exec" {
    inline = [
      "sudo hostname swarm-manager",
      "sudo apt-get install -y apt-transport-https ca-certificates curl software-properties-common",
      "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -",
      "sudo add-apt-repository \"deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable\"",
      "sudo apt-get update",
      "sudo apt-get install -y docker-ce=18.03.1~ce-0~ubuntu",
      "sudo usermod -aG docker ${var.user}",
      "echo $'{\n\t\"hosts\": [\"unix:///var/run/docker.sock\",\"tcp://0.0.0.0:2375\"]\n}' | sudo tee /etc/docker/daemon.json > /dev/null",
      "sudo sed -E 's/([$])//g' -i /etc/docker/daemon.json",
      "sudo chmod 400 /etc/docker/daemon.json",
      "sudo rm -rf /etc/systemd/system/docker.service.d && sudo mkdir -p /etc/systemd/system/docker.service.d",
      "echo $'[Service]\nExecStart=\nExecStart=/usr/bin/dockerd' | sudo tee /etc/systemd/system/docker.service.d/docker.conf > /dev/null",
      "sudo sed -E 's/([$])//g' -i /etc/systemd/system/docker.service.d/docker.conf",
      "sudo chmod 400 /etc/systemd/system/docker.service.d/docker.conf",
      "sudo systemctl daemon-reload",
      "sudo service docker restart",
      "sudo docker swarm init",
      "sudo docker info",
    ]
  }
}
