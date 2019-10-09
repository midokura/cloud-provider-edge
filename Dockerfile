FROM scratch
ARG BUILD_WORKDIR
ARG BINARY
LABEL maintainer "Miguel Herranz <miguel@midokura.com>"
COPY edge-cloud-controller-manager /
CMD ["/edge-cloud-controller-manager", "--cloud-provider", "edge"]
