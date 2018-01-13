---
layout: "docker"
page_title: "Docker: docker_service"
sidebar_current: "docs-docker-resource-service"
description: |-
  Manages the lifecycle of a Docker service.
---

# docker\_service

Manages the lifecycle of a Docker service.

## Example Usage

### Basic
```hcl
# Start a service
resource "docker_service" "foo_service" {
  name     = "swarm-foo-random"
  image    = "repo.mycompany.com:8080/foo-service"
  replicas = 10

  ports {
    internal = 80
    external = 8888
  }
}
```

### Advanced
```hcl
resource "docker_service" "service" {
  name     = "swarm-foo-random"
  image    = "repo.mycompany.com:8080/foo-service"
  replicas = "10"

  update_config {
    parallelism       = 2
    delay             = "10s"
    failure_action    = "pause"
    monitor           = "5s"
    max_failure_ratio = 0.1
    order             = "start-first"
  }

  rollback_config {
    parallelism       = 2
    delay             = "10s"
    failure_action    = "pause"
    monitor           = "5s"
    max_failure_ratio = 0.1
    order             = "start-first"
  }

  configs = [
    {
      config_id   = "${docker_config.service_config.id}"
      config_name = "${docker_config.service_config.name}"
      file_name   = "/root/configs/configs.json"
    },
  ]

  secrets = [
    {
      secret_id   = "${docker_secret.service_secret.id}"
      secret_name = "${docker_secret.service_secret.name}"
      file_name   = "/root/configs/secrets.json"
    },
  ]

  ports {
    internal = "${var.internal_port}"
    external = "${var.port}"
  }

  logging {
    driver_name = "awslogs"

    options {
      awslogs-region = "${var.aws_region}"
      awslogs-group  = "${var.env}/${var.service_name}"
    }
  }

  healthcheck {
    test     = ["CMD", "curl", "-f", "http://localhost:10000/${var.health_path}"]
    interval = "15s"
    timeout  = "10s"
    retries  = 4
  }
}
```

See also the `TestAccDockerService_full` test for a complete overview.

## Argument Reference

The following arguments are supported:

* `auth` - (Optional, block) See [Auth](#auth) below for details.
* `name` - (Required, string) The name of the Docker service.
* `image` - (Required, string) The image used to create the Docker service.
* `replicas` - (Optional, string) The image used to create the Docker service.
* `hostname` - (Optional, string) Hostname of the containers.
* `command` - (Optional, list of strings) The command to use to start the
    container. For example, to run `/usr/bin/myprogram -f baz.conf` set the
    command to be `["/usr/bin/myprogram", "-f", "baz.conf"]`.
* `env` - (Optional, set of strings) Environment variables to set.
* `host` - (Optional, list of strings) Each host is a string with the ip, the cononical hostname and its aliase serparated with a whitespace: `IP_address canonical_hostname [aliases...]` e.g. `10.10.10.10 host1`. 
* `network_mode` - (Optional, string) Network mode of the containers of the service (vip|dnsrr).
* `networks` - (Optional, set of strings) Id of the networks in which the
  container is.
* `mounts` - (Optional, set of blocks) See [Mounts](#mounts) below for details.
* `configs` - (Optional, set of blocks) See [Configs](#configs) below for details.
* `secrets` - (Optional, set of blocks) See [Secrets](#secrets) below for details.
* `ports` - (Optional, block) See [Ports](#ports) below for details.
* `update_config` - (Optional, block) See [UpdateConfig](#update-rollback-config) below for details.
* `rollback_config` - (Optional, block) See [RolbackConfig](#update-rollback-config) below for details.
* `constraints` - (Optional, set of strings) A set of constraints, e.g. `node=manager`.
* `logging` - (Optional, block) See [Logging](#logging) below for details.
* `healthcheck` - (Optional, block) See [Healthcheck](#healthcheck) below for details.

<a id="auth"></a>
### Auth

`auth` can be used additionally to the `registry_auth`. If both are given the `auth` wins and overwrites the auth of the provider.

* `server_address` - (Required, string) The address of the registry server
* `username` - (Optional, string) The username to use for authenticating to the registry. If this is blank, the `DOCKER_REGISTRY_USER` will also be checked. 
* `password` - (Optional, string) The password to use for authenticating to the registry. If this is blank, the `DOCKER_REGISTRY_PASS` will also be checked.

<a id="mounts"></a>
### Mounts

`mount` is a block within the configuration that can be repeated to specify
the extra mount mappings for the container. Each `mount` block supports
the following:

* `target` - (Required, string) The container path.
* `source` - (Required, string) The mount source (e.g. a volume name, a host path)
* `type` - (Required, string) The mount type: valid values are `bind`, `volume` or `tmpf`.
* `consistency` - (Optional, string) The consistency requirement for the mount: valid values are `default`, `consistent`, `cached` or `delegated`.
* `read_only` - (Optional, string) Whether the mount should be read-only
* `bind_propagation` - (Optional, string) Optional configuration for the `bind` type.
* `volume_no_copy` - (Optional, string) Optional configuration for the `volume` type - whether to populate volume with data from the target.
* `volume_labels` - (Optional, map of key/value pairs) Optional configuration for the `volume` type - adding labels.
* `volume_driver_name` - (Optional, string) Optional configuration for the `volume` type - the name of the driver to create the volume.
* `volume_driver_options` - (Optional, map of key/value pairs) Optional configuration for the `volume` type - options for the driver.
* `tmpf_size_bytes` - (Optional, int) Optional configuration for the `tmpf` type - The size for the tmpfs mount in bytes. 
* `tmpf_mode` - (Optional, int) Optional configuration for the `tmpf` type - The permission mode for the tmpfs mount in an integer.

<a id="configs"></a>
### Configs

`configs` is a block within the configuration that can be repeated to specify
the extra mount mappings for the container. Each `configs` block supports
the following:

* `config_id` - (Required, string) ConfigID represents the ID of the specific config.
* `config_name` - (Optional, string) The name of the config that this references, but internally it is just provided for lookup/display purposes
* `file_name` - (Optional, string) The specific target file that the config data is written within the docker container, e.g. `/root/config/config.json`

<a id="secrets"></a>
### Secrets

`secrets` is a block within the configuration that can be repeated to specify
the extra mount mappings for the container. Each `secrets` block supports
the following:

* `secret_id` - (Required, string) ConfigID represents the ID of the specific secret.
* `secret_name` - (Optional, string) The name of the secret that this references, but internally it is just provided for lookup/display purposes
* `file_name` - (Optional, string) The specific target file that the secret data is written within the docker container, e.g. `/root/secret/secret.json`

<a id="ports"></a>
### Ports

`ports` is a block within the configuration that can be repeated to specify
the port mappings of the container. Each `ports` block supports
the following:

* `internal` - (Required, int) Port within the container.
* `external` - (Required, int) Port exposed out of the container.
* `ip` - (Optional, string) IP address/mask that can access this port.
* `protocol` - (Optional, string) Protocol that can be used over this port,
  defaults to TCP.

<a id="update-rollback-config"></a>
### UpdateConfig and RollbackConfig

`update_config` or `rollback_config` is a block within the configuration that can be repeated only **once** to specify the extra update configuration for the containers of the service. The `update_config` `rollback_config` block supports the following:

* `parallelism` - (Optional, int) The maximum number of tasks to be updated in one iteration simultaneously (0 to update all at once).
* `delay` - (Optional, int) Delay between updates (ns|us|ms|s|m|h), e.g. 5s.
* `failure_action` - (Optional, int) Action on update failure: pause | continue | rollback.
* `monitor` - (Optional, int) Duration after each task update to monitor for failure (ns|us|ms|s|m|h)
* `max_failure_ratio` - (Optional, int) The failure rate to tolerate during an update.
* `order` - (Optional, int) Update order either 'stop-first' or 'start-first'.

<a id="logging"></a>
### Logging

`logging` is a block within the configuration that can be repeated only **once** to specify the extra logging configuration for the containers of the service. The `logging` block supports the following:

* `driver_name` - (Required, string) Either `none`, `json-file`, `syslog`, `journald`, `gelf`, `fluentd`, `awslogs`, `splunk`, `etwlogs` or `gcplogs`.
* `options` - (Optional, map of strings and strings) E.g.

```hcl
options {
  awslogs-region = "us-west-2"
  awslogs-group  = "dev/foo-service"
}
```

<a id="healthcheck"></a>
### Healthcheck

`logging` is a block within the configuration that can be repeated only **once** to specify the extra logging configuration for the containers of the service. The `logging` block supports the following:

* `test` - (Required, list of strings) Command to run to check health. For example, to run `curl -f http://localhost/health` set the
    command to be `["CMD", "curl", "-f", "http://localhost/health"]`.
* `interval` - (Optional, string) Time between running the check (ms|s|m|h). Default 10s.
* `timeout` - (Optional, string) Maximum time to allow one check to run (ms|s|m|h). Default 3s.
* `start_period` - (Optional, string) Start period for the container to initialize before counting retries towards unstable (ms|s|m|h). Default 2s.
* `retries` - (Optional, int) Consecutive failures needed to report unhealthy. Default 1.


## Attributes Reference

The following attributes are exported in addition to the above configuration:

* `id` (string)
