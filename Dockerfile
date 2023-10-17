# Builder image
FROM registry.access.redhat.com/ubi9/go-toolset:1.19 as builder
ENV GOPATH=$APP_ROOT
RUN env
COPY . .
RUN GOOS=linux GOARCH=amd64 GOFLAGS=-buildvcs=false go build -ldflags="-w -s" -o app github.com/konveyor/forklift-must-gather-api/pkg

# OpenShift CLI image (oc)
FROM quay.io/openshift/origin-cli:4.13 as ocimage

# Runner image
FROM registry.access.redhat.com/ubi9-minimal

LABEL name="konveyor/forklift-must-gather-api" \
      description="Konveyor Must Gather API service" \
      help="For more information visit https://github.com/konveyor/forklift-must-gather-api" \
      license="Apache License 2.0" \
      maintainer="maufart@redhat.com" \
      summary="Konveyor Must Gather API service" \
      url="https://quay.io/repository/kubev2v/forklift-must-gather-api" \
      usage="podman run konveyor/forklift-must-gather-api:latest" \
      com.redhat.component="forklift-must-gather-api-container" \
      io.k8s.display-name="must-gather-api" \
      io.k8s.description="Konveyor Must Gather API service" \
      io.openshift.expose-services="" \
      io.openshift.tags="operator,konveyor,forklift"

RUN microdnf -y install findutils && microdnf clean all

COPY --from=builder /opt/app-root/src/app /usr/bin/must-gather-api
COPY --from=ocimage /usr/bin/oc /usr/bin/oc

ENTRYPOINT ["/usr/bin/must-gather-api"]
