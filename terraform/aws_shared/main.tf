resource "aws_key_pair" "key_pair" {
  key_name   = "${var.project_name}-key-pair"
  public_key = file(var.ssh_public_key_path)

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-ssh-key-pair"
  }
}

output "key_name" {
  value = aws_key_pair.key_pair.key_name
}
