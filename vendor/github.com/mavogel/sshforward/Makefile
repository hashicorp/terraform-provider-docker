.PHONY: \
	all \
	lint \
	fmt \
	fmtcheck \
	pretest \
	test \
	clean

all: test

lint:
	@ go get -v github.com/golang/lint/golint
	[ -z "$$(golint . | grep -v 'type name will be used as docker.DockerInfo' | grep -v 'context.Context should be the first' | tee /dev/stderr)" ]

fmt:
	gofmt -s -w $$(go list ./... | grep -v vendor)

fmtcheck:
	[ -z "$$(gofmt -s -d $$(go list ./... | grep -v vendor) | tee /dev/stderr)" ]

pretest: lint fmtcheck

gotest:
	go test -race ./ -v -timeout 5m

test: pretest gotest

clean:
	go clean ./...
