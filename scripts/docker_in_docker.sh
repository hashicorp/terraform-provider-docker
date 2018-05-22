#!/bin/bash

IMAGE_NAME="mavogel/tf-test-docker:17.09-dind"
CONTAINER_NAME="tf-test-docker"
PROVIDER_NAME="terraform-provider-docker"

echo "#### Starting '$CONTAINER_NAME'"
# Optionally not mounting the code
#docker run --privileged --name "$CONTAINER_NAME" --rm -d "$IMAGE_NAME"
docker run --privileged --name "$CONTAINER_NAME" --rm -v "$GOPATH"/src/github.com/terraform-providers:/go/src/github.com/terraform-providers -d "$IMAGE_NAME" --storage-driver=overlay

# NOTE: although it is in the Dockerfile a `which openssl` does not find the binary
echo "#### Treating openssl"
docker exec -it "$CONTAINER_NAME" sh -c "cd /go/src/github.com/terraform-providers/${PROVIDER_NAME} && apk del openssl && apk add --no-cache openssl"

echo "#### INNER Docker version"
docker exec -it "$CONTAINER_NAME" sh -c "cd /go/src/github.com/terraform-providers/${PROVIDER_NAME} && docker version"

echo "#### Running tests..."
# Optionally checking the code out within the container instead of mounting it
#docker exec -it "$CONTAINER_NAME" sh -c "git clone https://github.com/terraform-providers/${PROVIDER_NAME}.git --depth=1 /go/src/github.com/terraform-providers/${PROVIDER_NAME}"
COMMANDS=(
  "test" 
  "testacc" 
  "vendor-status" 
  "vet"
  #"website-test" # some wget problems atm...
)
for cmd in "${COMMANDS[@]}";
  do docker exec -it "$CONTAINER_NAME" sh -c "cd /go/src/github.com/terraform-providers/${PROVIDER_NAME} && make $cmd"
done

echo "#### Stopping '$CONTAINER_NAME'"
docker stop "$CONTAINER_NAME"
