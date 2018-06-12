variable "aws_region" {
  description = "AWS region to launch servers."
  type        = "string"
  default     = "eu-central-1"
}

variable "ami" {
  description = "The ami to launch."
  type        = "string"
  default     = "ami-97e953f8"
}

variable "user" {
  description = "The user to use."
  type        = "string"
  default     = "ubuntu"
}

variable "host_key_name" {
  description = "The name of the host key file."
  type        = "string"
  default     = "host_key"
}

variable "instance_type" {
  description = "The type of ec2 instances to ramp up."
  type        = "string"
  default     = "t2.nano"
}
