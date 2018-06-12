output "bastion_host_ip" {
  value = "${module.aws.bastion_host_ip}"
}

output "swarm_host_public_ip" {
  value = "${module.aws.swarm_host_public_ip}"
}

output "swarm_host_private_ip" {
  value = "${module.aws.swarm_host_private_ip}"
}

output "service_id" {
  value = "${module.docker.service_id}"
}
