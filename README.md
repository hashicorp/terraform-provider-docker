Terraform Provider
==================

- Website: https://www.terraform.io
- [![Gitter chat](https://badges.gitter.im/hashicorp-terraform/Lobby.png)](https://gitter.im/hashicorp-terraform/Lobby)
- Mailing list: [Google Groups](http://groups.google.com/group/terraform-tool)

<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) 0.10.x
-	[Go](https://golang.org/doc/install) 1.8 (to build the provider plugin)

Building The Provider
---------------------

Clone repository to: `$GOPATH/src/github.com/terraform-providers/terraform-provider-$PROVIDER_NAME`

```sh
$ mkdir -p $GOPATH/src/github.com/terraform-providers; cd $GOPATH/src/github.com/terraform-providers
$ git clone git@github.com:terraform-providers/terraform-provider-$PROVIDER_NAME
```

Enter the provider directory and build the provider

```sh
$ cd $GOPATH/src/github.com/terraform-providers/terraform-provider-$PROVIDER_NAME
$ make build
```

Using the provider
----------------------
## Fill in for each provider

Developing the Provider
---------------------------

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (version 1.8+ is *required*). You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

To compile the provider, run `make build`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

```sh
$ make bin
...
$ $GOPATH/bin/terraform-provider-$PROVIDER_NAME
...
```

In order to test the provider, you can simply run `make test`.

```sh
$ make test
```

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
# run a single acceptance test: e.g. 'TestAccDockerRegistryImage_private' in 'data_source_docker_registry_image_test.go'
go test -v -timeout 30s github.com/terraform-providers/terraform-provider-docker/docker -run ^TestAccDockerRegistryImage_private$
```

In order to extend the provider and test it with `terraform`, check the latest version at https://releases.hashicorp.com/terraform-provider-$PROVIDER_NAME/ and raise it accordingly to semversion due to your changes in the code
```sh
$ export TF_PROVIDER_NEW_VERSION=0.1.2
$ go build -o terraform-provider-${PROVIDER_NAME}_v${TF_PROVIDER_NEW_VERSION}
# create ~/.terraform.d/plugins/'GOOS'_'GOARCH' with
# GOOS: darwin, freebsd, linux, and so on.
# GOARCH: 386, amd64, arm, s390x, and so on
# example for OSX:
$ mkdir -p ~/.terraform.d/plugins/darwin_amd64
$ mv -f terraform-provider-${PROVIDER_NAME}_v${TF_PROVIDER_NEW_VERSION} ~/.terraform.d/plugins/darwin_amd64
```

Add the explicit version of to your locally developed `terraform-provider-${PROVIDER_NAME}`:
```hcl
provider "docker" {
  version = "~> 0.1.2"
  ...
}
```

Don't forget to run `terraform init` each time you rebuild and moved your binary to the `plugins` folder.
