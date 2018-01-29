---
layout: "docker"
page_title: "Provider: Docker"
sidebar_current: "docs-docker-index"
description: |-
  The Docker provider is used to interact with Docker containers and images.
---

# Docker Provider

The Docker provider is used to interact with Docker containers and images.
It uses the Docker API to manage the lifecycle of Docker containers. Because
the Docker provider uses the Docker API, it is immediately compatible not
only with single server Docker but Swarm and any additional Docker-compatible
API hosts.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the Docker provider
provider "docker" {
  host = "tcp://127.0.0.1:2376/"
}

# Create a container
resource "docker_container" "foo" {
  image = "${docker_image.ubuntu.latest}"
  name  = "foo"
}

resource "docker_image" "ubuntu" {
  name = "ubuntu:latest"
}
```

## Registry Credentials

Registry credentials can be provided on a per-registry basis with the `registry_auth`
field, passing either a config file or the username/password directly.

-> **Note**
The location of the config file is on the machine terraform runs on, nevertheless if the specified docker host is on another machine.

``` hcl
provider "docker" {
  host = "tcp://localhost:2376"

  registry_auth {
    address = "registry.hub.docker.com"
    config_file = "~/.docker/config.json"
  }

  registry_auth {
    address = "quay.io:8181"
    username = "someuser"
    password = "somepass"
  }
}

data "docker_registry_image" "quay" {
  name = "myorg/privateimage"
}

data "docker_registry_image" "quay" {
  name = "quay.io:8181/myorg/privateimage"
}
```

-> **Note**
When passing in a config file make sure every repo in the `auths` object has
an `auth` string. If not you'll get an `ErrCannotParseDockercfg` by the underlying `go-dockerclient`. On OSX the `auth` base64 string is stored in the `osxkeychain`, but reading from there is not yet supported. See [go-dockerclient#677](https://github.com/fsouza/go-dockerclient/issues/677) for details. 

In this case, either use `username` and `password` directly or set the enviroment variables `DOCKER_REGISTRY_USER` and `DOCKER_REGISTRY_PASS` or add the string manually via

```sh
echo -n "user:pass" | base64
# dXNlcjpwYXNz=
``` 

and paste it into `~/.docker/config.json`:

```json
{
	"auths": {
		"repo.mycompany:8181": {
			"auth": "dXNlcjpwYXNz="
		}
	}	
}
```

## Bastion Host

A configuration to forward the Docker Daemon to a local address with an optional
Bastion host in between. For example if the Docker Daemon is in a private network which is only 
accessible via a Bastion host.

```hcl
provider "docker" {
  host      = "tcp://127.0.0.1:2376/"
  cert_path = "${pathexpand("~/.docker")}"
  
  forward_config {
    bastion_host                  = "10.0.0.1:22"
    bastion_host_user             = "bastionuser"
    bastion_host_private_key_file = "${pathexpand("~/.ssh/bastionkey)}"

    end_host                  = "20.0.0.1:22"
    end_host_user             = "endhostuser"
    end_host_private_key_file = "${pathexpand("~/.ssh/endhostkey)}"

    local_address  = "localhost:2376"
    remote_address = "localhost:2376"
  }
}
```

You can test before if the `forward` can be succesfully established with the following
command assuming you have a secured Docker Socket with `127.0.0.1` as authorized IP address
at the endhost on port 2376. It closes the forward after 10s of inactivity.

```sh
# setup the forward
$ ssh -f -L 2376:localhost:2376 \ 
    -o ExitOnForwardFailure=yes \ 
    -o ProxyCommand="ssh -l bastionuser -i ~/.ssh/bastionkey 10.0.0.1 -W %h:%p" \ 
    -o UserKnownHostsFile=/dev/null \ 
    -o StrictHostKeyChecking=no \ 
    -i ~/.ssh/endhostkey endhostuser@20.0.0.1 sleep 10
# access the forwarded docker host
$ docker --tlsverify -H 127.0.0.1:2376 info
```

## Argument Reference

The following arguments are supported:

* `host` - (Required) This is the address to the Docker host. If this is
  blank, the `DOCKER_HOST` environment variable will also be read.

* `cert_path` - (Optional) Path to a directory with certificate information
  for connecting to the Docker host via TLS. If this is blank, the
  `DOCKER_CERT_PATH` will also be checked.

* `ca_material`, `cert_material`, `key_material`, - (Optional) Content of `ca.pem`, `cert.pem`, and `key.pem` files
  for TLS authentication. Cannot be used together with `cert_path`.

* `registry_auth` - (Optional) A block specifying the credentials for a target
  v2 Docker registry.
   
  * `address` - (Required) The address of the registry.
 
  * `username` - (Optional) The username to use for authenticating to the registry.
  Cannot be used with the `config_file` option. If this is blank, the `DOCKER_REGISTRY_USER`
  will also be checked.
 
  * `password` - (Optional) The password to use for authenticating to the registry.
  Cannot be used with the `config_file` option. If this is blank, the `DOCKER_REGISTRY_PASS`
  will also be checked.
 
  * `config_file` - (Optional) The path to a config file containing credentials for
  authenticating to the registry. Cannot be used with the `username`/`password` options.
  If this is blank, the `DOCKER_CONFIG` will also be checked.
 
* `forward_config` - (Optional) Configuration to forward the Docker Daemon from a remote to a 
local address.
   
  * `bastion_host` - (Optional) The host address of the bastion host. A port for ssh needs
  to be specified explicitly, e.g. 10.0.0.1:22
  
  * `bastion_host_user` - (Optional) The user to login via ssh on the bastion host.
  
  * `bastion_host_password` - (Optional) The password of the user to login via ssh on the 
  bastion host. Cannot be used with the `bastion_host_private_key_file` option.
  
  * `bastion_host_private_key_file` - (Optional) The private key file associated with the user  
  to login via ssh on the bastion host. Cannot be used with the `bastion_host_password` option.
  
  * `end_host` - (Required) The host address of the end host where the Docker 
  Daemon is running. A port for ssh needs to be specified explicitly, e.g. 20.0.0.1:22.
  
  * `end_host_user` - (Optional) The user to login via ssh on the end host.
  
  * `end_host_password` - (Optional) The password of the user to login via ssh on the 
  end host. Cannot be used with the `end_host_private_key_file` option.
  
  * `end_host_private_key_file` - (Optional) The private key file associated with the user  
  to login via ssh on the end host. Cannot be used with the `end_host_password` option.
  
  * `local_address` - (Required) The local address the Docker Daemon is forwarded to.
  
  * `remote_address` - (Required) The address on the remote/end host the Docker Daemon is 
  forwarded from.

~> **NOTE on Certificates and `docker-machine`:**  As per [Docker Remote API
documentation](https://docs.docker.com/engine/reference/api/docker_remote_api/),
in any docker-machine environment, the Docker daemon uses an encrypted TCP
socket (TLS) and requires `cert_path` for a successful connection. As an alternative,
if using `docker-machine`, run `eval $(docker-machine env <machine-name>)` prior
to running Terraform, and the host and certificate path will be extracted from
the environment.
