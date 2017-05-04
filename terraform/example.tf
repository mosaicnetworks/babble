provider "aws" {
  access_key = "AKIAIREUC5KEUFOVU62Q"
  secret_key = "kaWJBy6LZBOS39NQX6/Cuk03Q5fM4+WkyKSCKJ9k"
  region     = "us-east-1"
}

resource "aws_instance" "example" {
  ami           = "ami-656be372"
  instance_type = "t1.micro"
}