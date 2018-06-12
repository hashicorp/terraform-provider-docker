# Docker behind a bastion host in a private network
This is an example setup to provision a `redis` as Docker service on a Docker Swarm which runs in a private network behind a bastion host on AWS.

## Important
- the setup is currently for the AWS region `eu-central-1`. Please adapt the `AMI` and `user` and the scripts below accordingly if you want to change the region.
- applying this example may cost you money. Please destroy it afterwards with `$ terraform destroy`
- the example has been split into 2 modules because the `Docker` provider needs to connect to the end host already during a `terraform plan`

## Setup
Create a new ssh keypair for this example:
```sh
$ mkdir ~/.ssh/tf-examples
$ ssh-keygen -b 4096 -t rsa -f ~/.ssh/tf-examples/host_key -q -N ""
```

Run `terraform` to apply this plan.
```sh
$ terraform init
$ terraform apply -target=module.aws
$ terraform apply -target=module.docker
```

## Useful commands
Connect to the bastion host:
```sh
$ ssh 
  -o StrictHostKeyChecking=no \
  -o UserKnownHostsFile=/dev/null \
  -i ~/.ssh/tf-examples/host_key \
  ubuntu@$(terraform output bastion_host_ip)
```

Connect via `ssh` to the end host:
```sh
ssh \ 
  -o ProxyCommand="ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i ~/.ssh/tf-examples/host_key -W %h:%p ubuntu@$(terraform output bastion_host_ip)" \
  -o StrictHostKeyChecking=no \
  -o UserKnownHostsFile=/dev/null \ 
  -i ~/.ssh/tf-examples/host_key \
  ubuntu@$(terraform output swarm_host_private_ip)
```

Establish a `forward` via the jump host to `localhost:10001`. The `forward` will
be terminated automatically after 10 seconds:
```sh
ssh -f -L 10001:localhost:2375 \ 
  -o ExitOnForwardFailure=yes \
  -o ProxyCommand="ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i ~/.ssh/tf-examples/host_key -W %h:%p ubuntu@$(terraform output bastion_host_ip)" \
  -o StrictHostKeyChecking=no \
  -o UserKnownHostsFile=/dev/null \ 
  -i ~/.ssh/tf-examples/host_key \
  ubuntu@$(terraform output swarm_host_private_ip) sleep 10
```

When the `forward` is established you can check the running `redis` service
with the following command:
```sh
$ docker -H localhost:10001 service inspect $(terraform output service_id)
```

## Destroy
Destroy the `docker` module first, then the `aws` module:
```sh
$ terraform destroy -target=module.docker
$ terraform destroy -target=module.aws
```

Clean up the keys
```sh
$ rm -rf ~/.ssh/tf-examples
```
