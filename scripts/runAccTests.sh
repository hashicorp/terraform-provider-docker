#!/bin/bash
set -e

PRIVATE_REGISTRY_ADDRESS="127.0.0.1"
PRIVATE_REGISTRY_PORT="15000"

log() {
  echo ""
  echo "##################################"
  echo "-------> $1"
  echo "##################################"
}

setup() {
  export DOCKER_REGISTRY_ADDRESS="${PRIVATE_REGISTRY_ADDRESS}:${PRIVATE_REGISTRY_PORT}"
  export DOCKER_REGISTRY_USER="testuser"
  export DOCKER_REGISTRY_PASS="testpwd"
  export DOCKER_PRIVATE_IMAGE="${PRIVATE_REGISTRY_ADDRESS}:${PRIVATE_REGISTRY_PORT}/tftest-service:v1"
  
  # Create private registry
  ## Create self signed certs
  mkdir -p "$(pwd)"/scripts/testing/certs
  openssl req \
    -newkey rsa:2048 \
    -nodes \
    -x509 \
    -days 365 \
    -subj "/C=US/ST=Denial/L=Springfield/O=Dis/CN=${PRIVATE_REGISTRY_ADDRESS}" \
    -keyout "$(pwd)"/scripts/testing/certs/registry_auth.key \
    -out "$(pwd)"/scripts/testing/certs/registry_auth.crt
  ## Create auth
  mkdir -p "$(pwd)"/scripts/testing/auth
  # Start registry
  docker run --rm --entrypoint htpasswd registry:2 -Bbn testuser testpwd > "$(pwd)"/scripts/testing/auth/htpasswd
  docker run -d -p ${PRIVATE_REGISTRY_PORT}:5000 --rm --name private_registry \
    -v "$(pwd)"/scripts/testing/auth:/auth \
    -e "REGISTRY_AUTH=htpasswd" \
    -e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" \
    -e "REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd" \
    -v "$(pwd)"/scripts/testing/certs:/certs \
    -e "REGISTRY_HTTP_TLS_CERTIFICATE=/certs/registry_auth.crt" \
    -e "REGISTRY_HTTP_TLS_KEY=/certs/registry_auth.key" \
    registry:2
  # wait a bit for travis...
  sleep 5
  # Login to private registry
  docker login -u testuser -p testpwd ${PRIVATE_REGISTRY_ADDRESS}:${PRIVATE_REGISTRY_PORT}
  # Build private images and push to private registry and remove locally
  for i in $(seq 1 3); do 
    docker build -t tftest-service "$(pwd)"/scripts/testing -f "$(pwd)"/scripts/testing/Dockerfile_v${i}
    docker tag tftest-service ${PRIVATE_REGISTRY_ADDRESS}:${PRIVATE_REGISTRY_PORT}/tftest-service:v${i}
    docker push ${PRIVATE_REGISTRY_ADDRESS}:${PRIVATE_REGISTRY_PORT}/tftest-service:v${i}
  done
  # Remove built images locally
  for i in $(seq 1 3); do 
    docker rmi ${PRIVATE_REGISTRY_ADDRESS}:${PRIVATE_REGISTRY_PORT}/tftest-service:v${i}
  done
  docker rmi tftest-service
}

run() {
  #TF_ACC=1 go test ./docker -v -timeout 120m
  
  # for a single test comment the previous line and uncomment the next line
  TF_LOG=INFO TF_ACC=1 go test -v github.com/terraform-providers/terraform-provider-docker/docker -run ^TestAccDockerService_full$ -timeout 360s
  
  # keep the return value for the scripts to fail and clean properly
  return $?
}

cleanup() {
  unset DOCKER_REGISTRY_ADDRESS DOCKER_REGISTRY_USER DOCKER_REGISTRY_PASS DOCKER_PRIVATE_IMAGE
  echo "### unsetted env ###"
  for p in $(docker container ls -f 'name=private_registry' -q); do docker stop $p; done
  echo "### stopped private registry ###"
  rm -f "$(pwd)"/scripts/testing/auth/htpasswd
  rm -f "$(pwd)"/scripts/testing/certs/registry_auth.*
  echo "### removed auth and certs ###"
  for resource in "container" "volume"; do
    for r in $(docker $resource ls -f 'name=tftest-' -q); do docker $resource rm -f "$r"; done
    echo "### removed $resource ###"
  done
  for resource in "config" "secret" "network"; do
    for r in $(docker $resource ls -f 'name=tftest-' -q); do docker $resource rm "$r"; done
    echo "### removed $resource ###"
  done
}

## main
log "setup" && setup 
log "run" && run || (log "cleanup on failure" && cleanup && exit 1)
log "cleanup" && cleanup
