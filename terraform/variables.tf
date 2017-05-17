variable "access_key" {}
variable "secret_key" {}

variable "vpc" {
  default = "vpc-7951fb10"
}

variable "ig" {
  default = "igw-2d03de44"
}

variable "servers" {
  default = 4
}

variable "key_name" {
  description = "SSH key name in your AWS account for AWS instances."
}

variable "key_path" {
  description = "Path to the private key specified by key_name."
}