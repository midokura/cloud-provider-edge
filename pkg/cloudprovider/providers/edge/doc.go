/*
Copyright 2019 Midokura

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Package edge provides an implementation of Kubernetes cloud provider that
is suitable for edge deployments.

Currently its only functionality is to manage the NAT port mappings
on deployments where a IGD (Internet Gateway Device) with UPnP enabled.

Specifically, it will use UPnP-IGD to create a port mapping for each
port of a load balancer service defined in the Kubernetes cluster that have
been annotated with the annotation:

	midokura.com/load-balancer-type: upnp-igd

For example, the following service:

	apiVersion: v1
	kind: Service
	metadata:
	  name: my-service
	  namespace: default
	  labels:
	    "midokura.com/load-balancer-type": "upnp-igd"
	spec:
	  selector:
	    app: my-app
	  ports:
	    - protocol: TCP
	      port: <external_port>     # External port (WAN port)
	      nodePort: <internal_port> # Internal port (LAN port)
	      targetPort: 80            # Pod port
	      name: http
	  type: LoadBalancer

will create the TCP port mapping:

  <external_ip>:<external_port> -> <internal_ip>:<internal_port>

The internal port is the service node port and the internal IP is the IP of
the first node available (usually non-master node).
*/
package edge
