variable "aws_region" {
  description = "AWS region to launch servers."
  type        = "string"
}

variable "ami" {
  description = "The ami to launch."
  type        = "string"
}

variable "user" {
  description = "The user to use."
  type        = "string"
}

variable "host_key_name" {
  description = "The name of the host key file."
  type        = "string"
}

variable "instance_type" {
  description = "The type of ec2 instances to ramp up."
  type        = "string"
}
