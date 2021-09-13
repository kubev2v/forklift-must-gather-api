# Builder image
FROM registry.access.redhat.com/ubi8/go-toolset:1.14.12 as builder
ENV GOPATH=$APP_ROOT
RUN env
COPY . .
RUN go build -o app github.com/konveyor/forklift-must-gather-api/pkg


# Runner image
FROM registry.access.redhat.com/ubi8-minimal

LABEL name="konveyor/forklift-must-gather-api" \
      description="Konveyor Must Gather API service" \
      help="For more information visit https://github.com/konveyor/forklift-must-gather-api" \
      license="Apache License 2.0" \
      maintainer="maufart@redhat.com" \
      summary="Konveyor Must Gather API service" \
      url="https://quay.io/repository/konveyor/forklift-must-gather-api" \
      usage="podman run konveyor/forklift-must-gather-api:latest" \
      com.redhat.component="forklift-must-gather-api-container" \
      io.k8s.display-name="must-gather-api" \
      io.k8s.description="Konveyor Must Gather API service" \
      io.openshift.expose-services="" \
      io.openshift.tags="operator,konveyor,forklift"

COPY --from=builder /opt/app-root/src/app /usr/bin/must-gather-api

# RUN microdnf -y install tar && microdnf clean all

ENTRYPOINT ["/usr/bin/must-gather-api"]
