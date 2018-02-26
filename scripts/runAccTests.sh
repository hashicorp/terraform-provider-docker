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
  # Is fixed in v18.02 https://github.com/moby/moby/issues/35933#issuecomment-366149721
  for c in $(docker container ls --filter=name=service- -aq); do docker rm -f $c; done
  echo "### removed stopped containers ###"
  for c in $(docker config ls --filter=name=myconfig -q); do docker config rm $c; done
  for s in $(docker secret ls --filter=name=mysecret -q); do docker secret rm $s; done
  echo "### configs and secrets ###"
  for i in $(docker images -aq 127.0.0.1:5000/my-private-service); do docker rmi -f $i; done
  echo "### removed my-private-service images ###"
}

## main
log "setup" && setup 
log "run" && run && echo $?
if [ $? -ne 0 ]; then
  log "cleanup" && cleanup
  exit 1
fi
log "cleanup" && cleanup
