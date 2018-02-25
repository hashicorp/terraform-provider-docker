#!/bin/bash
set -e

log() {
  echo ""
  echo "##################################"
  echo "-------> $1"
  echo "##################################"
}

setup() {
  export DOCKER_REGISTRY_ADDRESS="127.0.0.1:5000"
  export DOCKER_REGISTRY_USER="testuser"
  export DOCKER_REGISTRY_PASS="testpwd"
  export DOCKER_PRIVATE_IMAGE="127.0.0.1:5000/my-private-service:v1"
  sh scripts/testing/setup_private_registry.sh
}

run() {
  # Run the acc test suite
  TF_ACC=1 go test ./docker -v -timeout 120m
  
  # for a single test
  #TF_LOG=INFO TF_ACC=1 go test -v github.com/terraform-providers/terraform-provider-docker/docker -run ^TestAccDockerService_basic$ -timeout 360s
  # keep the return for the scripts to fail and clean properly
  return $?
}

cleanup() {
  unset DOCKER_REGISTRY_ADDRESS DOCKER_REGISTRY_USER DOCKER_REGISTRY_PASS DOCKER_PRIVATE_IMAGE
  echo "### unsetted env ###"
  docker stop private_registry
  echo "### stopped private registry ###"
  rm -f scripts/testing/auth/htpasswd
  rm -f scripts/testing/certs/registry_auth.*
  echo "### removed auth and certs ###"
  docker rm -f $(docker container ls --filter=name=service- -aq)
  echo "### removed stopped containers ###"
  docker config rm $(docker config ls --filter=name=myconfig- -q)
  docker secret rm $(docker secret ls --filter=name=mysecret- -q)
  echo "### configs and secrets ###"
  
  # fails due to https://github.com/moby/moby/issues/32620 because sometimes there a still running containers of 
  # removed services
  docker rmi -f $(docker images -aq 127.0.0.1:5000/my-private-service)
  echo "### removed my-private-service images ###"
}

## main
log "setup" && setup 
log "run" && run && echo $?
# travis fails from time to time there
[ "$TRAVIS" != "true" ]; log "cleanup" && cleanup
