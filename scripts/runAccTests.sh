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
  make testacc
  return $?
  # for a single test
  #TF_LOG=INFO TF_ACC=1 go test -v -timeout 120s github.com/terraform-providers/terraform-provider-docker/docker -run ^TestAccDockerService_basic$
}

cleanup() {
  unset DOCKER_REGISTRY_ADDRESS DOCKER_REGISTRY_USER DOCKER_REGISTRY_PASS DOCKER_PRIVATE_IMAGE
  rm -f scripts/testing/auth/htpasswd
  rm -f scripts/testing/certs/registry_auth.*
  docker stop private_registry
  # consider running this manually to clean up the
  # updateabe configs and secrets
  #docker config rm $(docker config ls -q)
  #docker secret rm $(docker secret ls -q)
}

## main
log "setup" && setup 
log "run" && run && echo $?
if [ $? -ne 0 ]; then
  log "cleanup" && cleanup 
  exit 1
fi
log "cleanup" && cleanup