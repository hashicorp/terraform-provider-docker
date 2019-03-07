---
layout: "docker"
page_title: "Docker: docker_secret"
sidebar_current: "docs-docker-resource-secret"
description: |-
  Manages the secrets of a Docker service in a swarm.
---

# docker\_secret

Manages the secrets of a Docker service in a swarm.

## Example Usage

### Basic

```hcl
# Creates a secret
resource "docker_secret" "foo_secret" {
  name = "foo_secret"
  data = "ewogICJzZXJsaasIfQo="
}
```

#### Update secret with no downtime
To update a `secret`, Terraform will destroy the existing resource and create a replacement. To effectively use a `docker_secret` resource with a `docker_service` resource, it's recommended to specify `create_before_destroy` in a `lifecycle` block. Provide a unique `name` attribute, for example
with one of the interpolation functions `uuid` or `timestamp` as shown
in the example below. The reason is [moby-35803](https://github.com/moby/moby/issues/35803).

```hcl
resource "docker_secret" "service_secret" {
  name = "${var.service_name}-secret-${replace(timestamp(),":", ".")}"
  data = "${base64encode(data.template_file.service_secret_tpl.rendered)}"

  lifecycle {
    ignore_changes        = ["name"]
    create_before_destroy = true
  }
}

resource "docker_service" "service" {
  # ...
  secrets = [
    {
      secret_id   = "${docker_secret.service_secret.id}"
      secret_name = "${docker_secret.service_secret.name}"
      file_name   = "/root/configs/configs.json"
    },
  ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required, string) The name of the Docker secret.
* `data` - (Required, string) The base64 encoded data of the secret.
* `labels` - (Optional, map of string/string key/value pairs) User-defined key/value metadata.

## Attributes Reference

The following attributes are exported in addition to the above configuration:

* `id` (string)
