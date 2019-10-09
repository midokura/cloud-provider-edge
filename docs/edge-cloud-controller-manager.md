# Edge Cloud Controller Manager

An external cloud controller manager for running kubernetes in edge clusters.

## Introduction

edge-cloud-controller-manager is a implementation of a external 'cloud' provider for edge clusters.

An external cloud provider is a kubernetes controller that runs cloud provider-specific loops required
for the functioning of kubernetes, but outside of the kubernetes `kube-controller-manager`.

As such, you must disable these controller loops in the `kube-controller-manager` if you are running the
`edge-cloud-controller-manager`. You can disable the controller loops by setting the `--cloud-provider`
flag to `external` when starting the kube-controller-manager.

For more details, please see:

- <https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/20180530-cloud-controller-manager.md>
- <https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/#running-cloud-controller-manager>
- <https://kubernetes.io/docs/tasks/administer-cluster/developing-cloud-controller-manager/>

## Installation

An example deployment to use in [k3s](https://github.com/rancher/k3s) can be installed using:

```
$ kubectl apply -f install/edge-cloud-controller-manager-k3s-deployment.yaml
```

It should be easily adapted to other Kubernetes clusters.

## Examples

Here are some examples of how you could leverage `openstack-cloud-controller-manager`:

- [loadbalancers](../examples/loadbalancers/)
