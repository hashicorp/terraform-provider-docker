output "bastion_host_ip" {
  value = "${aws_instance.bastion_host.public_ip}"
}

output "swarm_host_public_ip" {
  value = "${aws_instance.swarm_manager.public_ip}"
}

output "swarm_host_private_ip" {
  value = "${aws_instance.swarm_manager.private_ip}"
}
