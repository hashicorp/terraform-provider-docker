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

```hcl
# Creates a config
resource "docker_config" "foo_config" {
  name = "foo_config"
  data = "ewogICJzZXJIfQo="
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required, string) The name of the Docker config.
* `data` - (Required, string) The base64 encoded data of the config.


## Attributes Reference

The following attributes are exported in addition to the above configuration:

* `id` (string)
