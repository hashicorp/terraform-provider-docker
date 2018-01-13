---
layout: "docker"
page_title: "Docker: docker_secret"
sidebar_current: "docs-docker-resource-secret"
description: |-
  Manages the secrets of a Docker service in a swarm.
---

# docker\_secret

Manages the secreturation of a Docker service in a swarm.

## Example Usage

### Basic

```hcl
# Creates a secret
resource "docker_secret" "foo_secret" {
  name = "foo_secret"
  data = "ewogICJzZXJsaasIfQo="
}
```

### Advanced
See `docker_config` resource.

## Argument Reference

The following arguments are supported:

* `name` - (Required, string) The name of the Docker secret.
* `data` - (Required, string) The base64 encoded data of the secret.


## Attributes Reference

The following attributes are exported in addition to the above configuration:

* `id` (string)
