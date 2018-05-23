#!/bin/bash

IMAGE_NAME="mavogel/tf-test-docker:17.09-dind"
CONTAINER_NAME="tf-test-docker"
PROVIDER_NAME="terraform-provider-docker"

echo "#### Starting '$CONTAINER_NAME'"
docker run --privileged --name "$CONTAINER_NAME" --rm -v "$GOPATH"/src/github.com/terraform-providers:/go/src/github.com/terraform-providers -d "$IMAGE_NAME" --storage-driver=overlay

# NOTE: although it is in the Dockerfile a `which openssl` does not find the binary
echo "#### Fixing openssl"
docker exec -it "$CONTAINER_NAME" sh -c "cd /go/src/github.com/terraform-providers/${PROVIDER_NAME} && apk del openssl && apk add --no-cache openssl"

echo "#### INNER Docker version"
docker exec -it "$CONTAINER_NAME" sh -c "cd /go/src/github.com/terraform-providers/${PROVIDER_NAME} && docker version"

echo "#### Running tests..."
COMMANDS=(
  "test" 
  "testacc" 
  "vendor-status" 
  "vet"
  "website-test"
)
for cmd in "${COMMANDS[@]}";
  do docker exec -it "$CONTAINER_NAME" sh -c "cd /go/src/github.com/terraform-providers/${PROVIDER_NAME} && make $cmd"
done

echo "#### Stopping '$CONTAINER_NAME'"
docker stop "$CONTAINER_NAME"
