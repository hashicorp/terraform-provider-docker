.PHONY: \
	all \
	lint \
	vet \
	fmtcheck \
	pretest \
	test \
<<<<<<< HEAD
	integration
=======
	integration \
	clean
>>>>>>> Update dependencies. Mainly go-dockerclient.

all: test

lint:
<<<<<<< HEAD
	@ go get -v golang.org/x/lint/golint
	[ -z "$$(golint . | grep -v 'type name will be used as docker.DockerInfo' | grep -v 'context.Context should be the first' | tee /dev/stderr)" ]

vet:
	go vet ./...

fmtcheck:
	[ -z "$$(gofmt -s -d *.go ./testing | tee /dev/stderr)" ]
=======
	@ go get -v github.com/golang/lint/golint
	[ -z "$$(golint . | grep -v 'type name will be used as docker.DockerInfo' | grep -v 'context.Context should be the first' | tee /dev/stderr)" ]

vet:
	go vet $$(go list ./... | grep -v vendor)

fmt:
	gofmt -s -w $$(go list ./... | grep -v vendor)

fmtcheck:
	[ -z "$$(gofmt -s -d $$(go list ./... | grep -v vendor) | tee /dev/stderr)" ]
>>>>>>> Update dependencies. Mainly go-dockerclient.

testdeps:
	go get -u github.com/golang/dep/cmd/dep
	dep ensure -v

pretest: testdeps lint vet fmtcheck

gotest:
<<<<<<< HEAD
	go test -race ./...
=======
	go test -race $$(go list ./... | grep -v vendor)
>>>>>>> Update dependencies. Mainly go-dockerclient.

test: pretest gotest

integration:
	go test -tags docker_integration -run TestIntegration -v
<<<<<<< HEAD
=======

clean:
	go clean ./...
>>>>>>> Update dependencies. Mainly go-dockerclient.
