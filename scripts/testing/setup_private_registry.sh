#!/bin/bash
set -e

# Create private registry
## Create self signed certs
mkdir -p scripts/testing/certs
openssl req \
  -newkey rsa:2048 \
  -nodes \
  -x509 \
  -days 365 \
  -subj "/C=US/ST=Denial/L=Springfield/O=Dis/CN=127.0.0.1" \
  -keyout scripts/testing/certs/registry_auth.key \
  -out scripts/testing/certs/registry_auth.crt
## Create auth
mkdir -p scripts/testing/auth
# Start registry
docker run --rm --entrypoint htpasswd registry:2 -Bbn testuser testpwd > scripts/testing/auth/htpasswd
docker run -d -p 5000:5000 --rm --name private_registry \
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
docker login -u testuser -p testpwd 127.0.0.1:5000
# Build private images
for i in $(seq 1 3); do 
  docker build -t tftest-service ./scripts/testing -f ./scripts/testing/Dockerfile_v${i}
  docker tag tftest-service 127.0.0.1:5000/tftest-service:v${i}
  docker push 127.0.0.1:5000/tftest-service:v${i}
done
