# Load balancers

Edge Cloud Controller Manager runs service controller,
which is responsible for watching services of type ```LoadBalancer```
and creating "load balancers" to satisfy its requirements: currently
this translates to setup NATP port mappings in the edge Internet gateway
device using UPnP IGD protocol.

Here are some examples of how it's used.

## UPnP IGD 'load balancer'

When you create a service with ```type: LoadBalancer``` and annotate it
with ```midokura.com/load-balancer-type: upnp-igd```, the edge cloud
controller manager will use the UPnP IGD mechanism to setup the required
port mappings in the Internet gateway device.

If this annotation is not specified, the edge controller will ignore the service
and no port mappings will be done.

NOTE: The Edge Cloud Controller Manager (namely its embedded UPnP client)
should run on one of nodes on which the Node Port of the service is exposed.
See the documentation of ``LegacyNodeRoleBehavior`` and
``ServiceNodeExclusion``
[feature gates](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/) and tweak your deployment of
the Edge Cloud Controller Manager accordingly.

For example:

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: http-nginx-deployment
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - containerPort: 80
---
kind: Service
apiVersion: v1
metadata:
  name: http-nginx-service
  annotations:
    midokura.com/load-balancer-type: upnp-igd
spec:
  selector:
    app: nginx
  type: LoadBalancer
  ports:
  - name: http
    port: 8080
    targetPort: 80
```

These definition is included in a file in `examples/` folder. We can use it
to deploy in a Kuberneter cluster:

```bash
$ kubectl create -f examples/loadbalancers/upnp-igd-http-nginx.yaml.yaml
```

Watch the service and await an ```EXTERNAL-IP``` by the following command.
This will be the load balancer IP which you can use to connect to your service.

```bash
$ watch kubectl get service
NAME                 CLUSTER-IP     EXTERNAL-IP       PORT(S)        AGE
http-nginx-service   10.0.0.10      122.112.219.229   80:30000/TCP   5m
```

This should have created a TCP port mapping in the Internet gateway device,
using ```<cluster-name>/<service-namespace>/<service-name>/<port-name>``` as
description.

Usually that can be checked using the web configuration interface of the device
or by using a UPnP client, like [miniupnpc](http://miniupnp.free.fr/):

```
$ upnpc -l
upnpc : miniupnpc library test client. (c) 2005-2014 Thomas Bernard
Go to http://miniupnp.free.fr/ or http://miniupnp.tuxfamily.org/
for more information.
List of UPNP devices found on the network :
 desc: http://192.168.1.1:49536/5cdae2e3/IGDV1/rootDesc.xml
 st: urn:schemas-upnp-org:device:InternetGatewayDevice:1

 desc: http://192.168.1.1:49536/5cdae2e3/rootDesc.xml
 st: urn:schemas-upnp-org:device:InternetGatewayDevice:1

Found valid IGD : http://192.168.1.1:49536/5cdae2e3/ctl/IPConn
Local LAN ip address : 192.168.1.10
Connection Type : IP_Routed
Status : Connected, uptime=1160775s, LastConnectionError : ERROR_NONE
  Time started : Thu Sep 26 06:37:46 2019
MaxBitRateDown : 1000000000 bps (1000.0 Mbps)   MaxBitRateUp 1000000000 bps (1000.0 Mbps)
ExternalIPAddress = 122.112.219.229
 i protocol exPort->inAddr:inPort description remoteHost leaseTime
 0 TCP  8080->192.168.1.10:30000 'kubernetes/default/http-nginx-service/http' '' 0
GetGenericPortMappingEntry() returned 713 (SpecifiedArrayIndexInvalid)
```

You can now access your service from Internet:

```bash
$ curl http://122.112.219.229:8080
```

Note that to access it from the same LAN, the Internet gateway device must support
[NAT loopback](https://en.wikipedia.org/wiki/Network_address_translation#NAT_loopback)
 (also known as [hairpinning](https://en.wikipedia.org/wiki/Hairpinning)).


