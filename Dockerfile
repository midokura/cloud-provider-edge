FROM scratch
ARG BUILD_WORKDIR
ARG BINARY
LABEL maintainer "Miguel Herranz <miguel@midokura.com>"

# Adding /tmp directory: it is needed to generate self-signed cert file /tmp/client-ca-file###
# See https://github.com/kubernetes/apiserver/blob/dd282eb3a3000bb8b94afe3be485c6e5647e4409/pkg/server/options/authentication.go#L353)
WORKDIR /tmp

COPY edge-cloud-controller-manager /
CMD ["/edge-cloud-controller-manager", "--cloud-provider", "edge", "--cloud-config", "/dev/null", "--vmodule=edge*=5", "--feature-gates", "LegacyNodeRoleBehavior=false"]

# Note: --feature-gates='LegacyNodeRoleBehavior=false' is needed due to master not included in nodes able to provide load balancing.
#       See https://github.com/kubernetes/kubernetes/blob/37c3a4da97a866a863eb71543a79a56e9834da14/pkg/controller/service/service_controller.go#L642
