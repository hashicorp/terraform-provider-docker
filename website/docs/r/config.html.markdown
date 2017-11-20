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
In this example you can use the `${var.foo_port}` variable to dynamically
set the `${port}` variable in the `foo.configs.json.tpl` template and create
the data of the `foo_config` with the help of the `base64encode` interpolation 
function.

`foo.config.json.tpl`
```json
{
  "server": {
    "public_port": ${port}
  }
}
```

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

## Argument Reference

The following arguments are supported:

* `name` - (Required, string) The name of the Docker config.
* `data` - (Required, string) The base64 encoded data of the config.


## Attributes Reference

The following attributes are exported in addition to the above configuration:

* `id` (string)
