variable "bastion_host_public_ip" {
  description = "The public ip of the bastion host."
  type        = "string"
}

variable "swarm_manager_private_ip" {
  description = "The private ip of the swarm manager host."
  type        = "string"
}

variable "user" {
  description = "The user to use."
  type        = "string"
}

variable "host_key_name" {
  description = "The name of the host key file."
  type        = "string"
  default     = "host_key"
}
