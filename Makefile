# Image URL to use all building/pushing image targets
IMG ?= quay.io/konveyor/forklift-must-gather-api:latest
GOOS ?= `go env GOOS`
GOBIN ?= ${GOPATH}/bin
GO111MODULE = auto

ci: all

all: test manager

# Run tests
test: fmt vet
	go test ./pkg/... -coverprofile cover.out

# Build manager binary
manager: fmt vet
	go build -o bin/app github.com/konveyor/forklift-must-gather-api/pkg/must-gather-api

# Run against the configured Kubernetes cluster in ~/.kube/config
run: fmt vet
	go run ./pkg/must-gather-api.go

# Run go fmt against code
fmt:
	go fmt ./pkg/...

# Run go vet against code
vet:
	go vet ./pkg/...

# Build the docker image
#docker-build: test
docker-build:
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}
