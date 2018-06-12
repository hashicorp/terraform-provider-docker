module "aws" {
  source        = "./aws"
  aws_region    = "${var.aws_region}"
  ami           = "${var.ami}"
  user          = "${var.user}"
  host_key_name = "${var.host_key_name}"
  instance_type = "${var.instance_type}"
}

module "docker" {
  source                   = "./docker"
  bastion_host_public_ip   = "${module.aws.bastion_host_ip}"
  swarm_manager_private_ip = "${module.aws.swarm_host_private_ip}"
  user                     = "${var.user}"
  host_key_name            = "${var.host_key_name}"
}
