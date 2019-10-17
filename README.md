# cloud-provider-edge

This project aims to provide an Kubernetes 'cloud provider' for edge computing.

It provides the interface between a Kubernetes cluster and other services
usually available in edge scenarios.

These often means nodes behind a home or corporate network behind a router
acting as Internet gateway device (IGD), like a DSL or optic fiber router.

The functionality provided is limited, but it is expected to evolve to cover
more cases.

# Features

 * Provision, monitor and remove Intenet gateway port mappings necessary to
   allow Kubernetes services to be reached from Internet.

# Usage

Check [edge-cloud-controller-manager](docs/edge-cloud-controller-manager.md) and
[examples](examples/loadbalancers/README.md)

# Roadmap

With no particular order:

 * Add more documentation
 * Add tests
 * Test against more scenarios and gateways devices.
 * Add more [cloud provider interfaces](https://github.com/kubernetes/cloud-provider/blob/master/cloud.go#L42-L62)
   aside of LoadBalancer.

# References

* https://kubernetes.io/docs/tasks/administer-cluster/developing-cloud-controller-manager/
* https://github.com/GlenDC/go-external-ip
* https://github.com/huin/goupnp

## Acknownledgements

Some code structure and documentation were adapted from other cloud providers, like:

 * [OpenStack](https://github.com/kubernetes/cloud-provider-openstack)
 * [AWS](https://github.com/kubernetes/cloud-provider-aws)
