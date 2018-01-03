---
layout: "docker"
page_title: "Docker: docker_config"
sidebar_current: "docs-docker-resource-config"
description: |-
  Manages the configs of a Docker service in a swarm.
---

# docker\_config

Manages the configuration of a Docker service in a swarm.

## Example Usage

## Basic
```hcl
# Creates a config
resource "docker_config" "foo_config" {
  name = "foo_config"
  data = "ewogICJzZXJIfQo="
}
```

### Advanced
#### Dynamically set config with a template
In this example you can use the `${var.foo_port}` variable to dynamically
set the `${port}` variable in the `foo.configs.json.tpl` template and create
the data of the `foo_config` with the help of the `base64encode` interpolation 
function.

File `foo.config.json.tpl`

```json
{
  "server": {
    "public_port": ${port}
  }
}
```

File `main.tf`

```hcl
# Creates the template in renders the variable
data "template_file" "foo_config_tpl" {
  template = "${file("foo.config.json.tpl")}"

  vars {
    port = "${var.foo_port}"
  }
}

# Creates the config
resource "docker_config" "foo_config" {
  name = "foo_config"
  data = "${base64encode(data.template_file.foo_config_tpl.rendered)}"
}
```

#### Update config with no downtime
This example shows how a config can be updated and the service has not to be shutdown.
Each config gets a unique timestamp. The drawback is that for every update a new config will be created. This is because at the moment
it is not possible to update the data for a config (or secret). The issue [moby-35803](https://github.com/moby/moby/issues/35803)
will solve this in the future.

```hcl
resource "docker_config" "service_config" {
  name = "${var.service_name}-config-${replace(timestamp(),":", ".")}"
  data = "${base64encode(data.template_file.service_config_tpl.rendered)}"

  lifecycle {
    ignore_changes = ["name"]
  }
}
resource "docker_service" "service" {
   # ...
   configs = [
    {
      config_id   = "${docker_config.service_config.id}"
      config_name = "${docker_config.service_config.name}"
      file_name   = "/root/configs/configs.json"
    },
  ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required, string) The name of the Docker config.
* `data` - (Required, string) The base64 encoded data of the config.


## Attributes Reference

The following attributes are exported in addition to the above configuration:

* `id` (string)
