# Docker service
provider "docker" {
  version = "~> 1.0.1"

  host = "tcp://localhost:10001/"

  forward_config {
    bastion_host                  = "${var.bastion_host_public_ip}:22"
    bastion_host_user             = "${var.user}"
    bastion_host_private_key_file = "${pathexpand("~/.ssh/tf-examples/${var.host_key_name}")}"

    end_host                  = "${var.swarm_manager_private_ip}:22"
    end_host_user             = "${var.user}"
    end_host_private_key_file = "${pathexpand("~/.ssh/tf-examples/${var.host_key_name}")}"

    local_address  = "localhost:10001"
    remote_address = "localhost:2375"
  }
}

resource "docker_service" "foo" {
  name = "redis-terraform"

  mode {
    replicated {
      replicas = "2"
    }
  }

  task_spec {
    container_spec {
      image = "redis:3.0.6"
    }
  }
}
