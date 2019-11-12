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

package edge

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"strings"

	k8s "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog"

	"github.com/glendc/go-external-ip"
	"github.com/huin/goupnp/dcps/internetgateway2"
)

const (
	// LoadBalancerTypeAnnotation annotates load balancer type
	LoadBalancerTypeAnnotation string = "midokura.com/load-balancer-type"
)

// LoadBalancerTypeAnnotation values
const (
	// UniversalPlugAndPlayInternetGatewayDeviceLoadBalancerType UPnP IGD load balancer type
	UniversalPlugAndPlayInternetGatewayDeviceLoadBalancerType = "upnp-igd"
)

type ensureOrUpdate bool

const (
	ensure ensureOrUpdate = true
	update ensureOrUpdate = false
)

type isDeleteOrIsNotDelete bool

const (
	isDelete    isDeleteOrIsNotDelete = true
	isNotDelete isDeleteOrIsNotDelete = false
)

type portMappingLeaseDuration uint16

const (
	infinitePortMappingLeaseDuration portMappingLeaseDuration = 0
)

// portMapping store the data for a port mapping
type portMapping struct {
	servicePort k8s.ServicePort
	nodeIP      string
}

// loadBalancer store the data for a Kubernetes load balancer
type loadBalancer struct {
	portMappings []portMapping
	status       *k8s.LoadBalancerStatus // basically to store ingress IP address
}

type clientInterface interface {
	AddPortMapping(host string, externalPort uint16, proto string, internalPort uint16, internalIP string, enabled bool, desc string, lease uint32) error
	DeletePortMapping(host string, externalPort uint16, proto string) error
}

// LoadBalancer store the data for Load Balancer API Edge cloud provider
type LoadBalancer struct {
	// UPnP IGD WANIP connection client
	client clientInterface
	// Local address of the client on the interface towards the UPnP device.
	// Initialized on first call to client()
	localAddress net.IP
	// External IP is the IP of the public side of the NATP mappings
	externalIP net.IP
	// List of known active load balancers
	loadBalancers map[string]loadBalancer
}

// NewLoadBalancer setup internal fields of LoadBalancer
func NewLoadBalancer() (*LoadBalancer, error) {
	lb := &LoadBalancer{}
	// get UPnP client
	clients, suberrors, err := internetgateway2.NewWANIPConnection2Clients()
	if err != nil {
		klog.Errorf("NewLoadBalancer: client error (with %d suberrors): %v", len(suberrors), err)
		for i, suberror := range suberrors {
			klog.Errorf("NewLoadBalancer: client suberror #%d of %d: %v", i, len(suberrors), suberror)
		}
		return nil, err
	}
	if len(clients) < 1 {
		klog.Errorf("NewLoadBalancer: no clients available")
		return nil, fmt.Errorf("no clients available")
	}
	if len(clients) > 1 {
		klog.Warningf("NewLoadBalancer: client warning: more than one client: using client #0 of %d (maybe wrongly!)", len(clients))
		for i, client := range clients {
			klog.Warningf("NewLoadBalancer: client #%d of %d: %v", i, len(clients), client)
		}
	}
	client := clients[0]
	lb.client = client
	// get local address
	lb.localAddress = getLocalAddressToHost(client.ServiceClient.Location.Hostname())
	// get external address
	externalIP, err := getExternalIP()
	if err != nil {
		klog.Errorf("NewLoadBalancer: error: external IP: %v", err)
		return nil, err
	}
	lb.externalIP = externalIP
	// init map
	lb.loadBalancers = make(map[string]loadBalancer)
	klog.Infof("NewLoadBalancer: addresses: {local: %s, external: %s}", lb.localAddress.String(), lb.externalIP.String())
	return lb, nil
}

///////////////////////////////////////////////////////////////////////////////
// API SECTION BEGIN
///////////////////////////////////////////////////////////////////////////////

// GetLoadBalancer returns whether the specified load balancer exists, and
// if so, what its status is.
// Implementations must treat the *v1.Service parameter as read-only and not modify it.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (lb *LoadBalancer) GetLoadBalancer(ctx context.Context, clusterName string, service *k8s.Service) (status *k8s.LoadBalancerStatus, exists bool, err error) {
	name := lb.GetLoadBalancerName(ctx, clusterName, service)
	if loadBalancer, exists := lb.loadBalancers[name]; exists {
		return loadBalancer.status, true, nil
	}
	return nil, false, nil
}

// GetLoadBalancerName returns the name of the load balancer. Implementations must treat the
// *v1.Service parameter as read-only and not modify it.
func (lb *LoadBalancer) GetLoadBalancerName(ctx context.Context, clusterName string, service *k8s.Service) string {
	return fmt.Sprintf("%s/%s/%s", clusterName, service.Namespace, service.Name)
}

// EnsureLoadBalancer creates a new load balancer 'name', or updates the existing one. Returns the status of the balancer
// Implementations must treat the *v1.Service and *v1.Node
// parameters as read-only and not modify them.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (lb *LoadBalancer) EnsureLoadBalancer(ctx context.Context, clusterName string, service *k8s.Service, nodes []*k8s.Node) (*k8s.LoadBalancerStatus, error) {
	loadBalancer, err := lb.ensureOrUpdateLoadBalancer(ctx, clusterName, service, nodes, ensure, isNotDelete)
	if err != nil {
		klog.Errorf("%v", err)
		return nil, err
	}
	return loadBalancer.status, nil
}

// UpdateLoadBalancer updates hosts under the specified load balancer.
// Implementations must treat the *v1.Service and *v1.Node
// parameters as read-only and not modify them.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (lb *LoadBalancer) UpdateLoadBalancer(ctx context.Context, clusterName string, service *k8s.Service, nodes []*k8s.Node) error {
	_, err := lb.ensureOrUpdateLoadBalancer(ctx, clusterName, service, nodes, update, isNotDelete)
	if err != nil {
		klog.Errorf("%v", err)
		return err
	}
	klog.Errorf("Address in update %v", &lb)
	return nil
}

// EnsureLoadBalancerDeleted deletes the specified load balancer if it
// exists, returning nil if the load balancer specified either didn't exist or
// was successfully deleted.
// This construction is useful because many cloud providers' load balancers
// have multiple underlying components, meaning a Get could say that the LB
// doesn't exist even if some part of it is still laying around.
// Implementations must treat the *v1.Service parameter as read-only and not modify it.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (lb *LoadBalancer) EnsureLoadBalancerDeleted(ctx context.Context, clusterName string, service *k8s.Service) error {
	_, err := lb.ensureOrUpdateLoadBalancer(ctx, clusterName, service, nil, ensure, isDelete)
	if err != nil {
		klog.Errorf("%v", err)
		return err
	}
	return nil
}

///////////////////////////////////////////////////////////////////////////////
// API SECTION END
///////////////////////////////////////////////////////////////////////////////

func (lb *LoadBalancer) ensureOrUpdateLoadBalancer(ctx context.Context, clusterName string, service *k8s.Service, nodes []*k8s.Node,
	ensure ensureOrUpdate, isDelete isDeleteOrIsNotDelete) (*loadBalancer, error) {
	err := lb.validateParametersOfLoadBalancer(ctx, clusterName, service, nodes)
	if err != nil {
		return nil, err
	}
	nodeIP := ""
	if !isDelete {
		clientIP := lb.localAddress.String()
		nodeIP, err = getFirstNodeInternalIP(clientIP, nodes)
		if err != nil {
			return nil, err
		}
	}
	name := lb.GetLoadBalancerName(ctx, clusterName, service)
	// getting current load balancer
	oldLoadBalancer, oldExisted := lb.loadBalancers[name]
	if !oldExisted {
		if !ensure {
			return nil, fmt.Errorf("cannot update load balancer '%s': not found", name)
		}
		if isDelete {
			oldLoadBalancer = newLoadBalancerWithPortMappings(service, nodeIP /* is "" */, lb.externalIP.String()) // for delete, assume unknown state is all installed (on a potentially unknown nodeIP, it shouldn't matter)
		} else {
			oldLoadBalancer = newLoadBalancerWithoutPortMappings(lb.externalIP.String()) // for not delete (create), assume unknown state is nothing installed
		}
	}
	// getting target load balancer
	var newLoadBalancer loadBalancer
	if isDelete {
		newLoadBalancer = newLoadBalancerWithoutPortMappings(lb.externalIP.String()) // for delete, target state is nothing installed
	} else {
		newLoadBalancer = newLoadBalancerWithPortMappings(service, nodeIP, lb.externalIP.String()) // for not delete (create), target state is all installed
	}
	// move from old to new
	err = lb.patchLoadBalancer(name, oldLoadBalancer.portMappings, newLoadBalancer.portMappings)
	if err != nil {
		return nil, err
	}
	// update load balancer map
	if bool(isDelete) && oldExisted {
		delete(lb.loadBalancers, name)
		return nil, nil
	}
	lb.loadBalancers[name] = newLoadBalancer
	return &newLoadBalancer, nil
}

func newLoadBalancerWithoutPortMappings(externalIP string) loadBalancer {
	lb := loadBalancer{
		portMappings: make([]portMapping, 0),
		status: &k8s.LoadBalancerStatus{
			Ingress: []k8s.LoadBalancerIngress{{IP: externalIP}},
		},
	}
	return lb
}

func newLoadBalancerWithPortMappings(service *k8s.Service, nodeIP string, externalIP string) loadBalancer {
	lb := loadBalancer{
		portMappings: make([]portMapping, len(service.Spec.Ports)),
		status: &k8s.LoadBalancerStatus{
			Ingress: []k8s.LoadBalancerIngress{{IP: externalIP}},
		},
	}
	for i, servicePort := range service.Spec.Ports {
		lb.portMappings[i].servicePort = servicePort
		lb.portMappings[i].nodeIP = nodeIP
	}
	return lb
}

func (lb *LoadBalancer) patchLoadBalancer(prefix string, old, new []portMapping) error {
	// create 'portMappingsToAdd' map from 'new.PortMappings'
	toBeAddedPortMappings := make(map[portMapping]bool)
	for _, portMapping := range new {
		toBeAddedPortMappings[portMapping] = true
	}

	// iterate old port mappings ...
	for _, portMapping := range old {
		if _, exists := toBeAddedPortMappings[portMapping]; exists { // ... if one already in new ...
			delete(toBeAddedPortMappings, portMapping) // ... remove it from 'to be added' set
		} else {
			err := lb.deletePortMapping(&portMapping)
			if err != nil {
				return err
			}
		}
	}

	// iterate to be added list and add them
	for portMapping := range toBeAddedPortMappings {
		err := lb.addPortMapping(prefix, &portMapping)
		if err != nil {
			return err
		}
	}
	return nil
}

func (lb *LoadBalancer) validateParametersOfLoadBalancer(ctx context.Context, clusterName string, service *k8s.Service, nodes []*k8s.Node) error {
	loadBalancerName := lb.GetLoadBalancerName(ctx, clusterName, service)
	errCtx := fmt.Sprintf("%s: %s", fname(), loadBalancerName)
	lbType, ok := service.Annotations[LoadBalancerTypeAnnotation]
	if !ok {
		// TODO: don't return error, just log
		klog.Infof("%s: missing '%s' annotation", errCtx, LoadBalancerTypeAnnotation)
		return fmt.Errorf("%s: missing '%s' annotation", errCtx, LoadBalancerTypeAnnotation)
	}
	if lbType != UniversalPlugAndPlayInternetGatewayDeviceLoadBalancerType {
		// TODO: don't return error, just log
		return fmt.Errorf("%s: unssuported load balancer type (annotation '%s=%s')", errCtx, LoadBalancerTypeAnnotation, lbType)
	}
	if service.Spec.Type != k8s.ServiceTypeLoadBalancer {
		return fmt.Errorf("%s: ServiceType must be '%s'", errCtx, k8s.ServiceTypeLoadBalancer)
	}
	if service.Spec.ClusterIP == k8s.ClusterIPNone {
		return fmt.Errorf("%s: ClusterIP must not be '%s'", errCtx, k8s.ClusterIPNone)
	}
	if service.Spec.PublishNotReadyAddresses != false {
		return fmt.Errorf("%s: PublishNotReadyAddresses must be false", errCtx)
	}
	if service.Spec.IPFamily != nil && *service.Spec.IPFamily != k8s.IPv4Protocol {
		return fmt.Errorf("%s: IPFamily must be %v: IPFamily '%v' not supported", errCtx, k8s.IPv4Protocol, service.Spec.IPFamily)
	}
	if service.Spec.SessionAffinity != k8s.ServiceAffinityNone {
		return fmt.Errorf("%s: SessionAffinity must be %s: SessionAffinity '%s' not supported", errCtx, k8s.ServiceAffinityNone, service.Spec.SessionAffinity)
	}
	if service.Spec.LoadBalancerIP != "" {
		klog.Warningf("%s: ignoring LoadBalancerIP: '%s'", errCtx, service.Spec.LoadBalancerIP)
	}
	if len(service.Spec.LoadBalancerSourceRanges) > 0 {
		// TODO: implement basic source range restriction functionality
		// See https://kubernetes.io/docs/tasks/access-application-cluster/configure-cloud-provider-firewall/
		klog.Warningf("%s: security warning: ignoring load balancer source range restrictions (not implemented): clients may connect from any IP address", errCtx)
	}
	for _, port := range service.Spec.Ports {
		if port.Protocol != k8s.ProtocolTCP && port.Protocol != k8s.ProtocolUDP {
			return fmt.Errorf("%s: port mapping for port %s: unsupported protocol %s", errCtx, port.Name, port.Protocol)
		}
		if port.TargetPort.Type == intstr.String {
			return fmt.Errorf("%s: port mapping for port %s: unsupported TargetPort type String", errCtx, port.Name)
		}
		if port.NodePort == 0 {
			return fmt.Errorf("%s: port mapping for port %s: a valid NodePort must be declared", errCtx, port.Name)
		}
	}
	return nil
}

func (lb *LoadBalancer) addPortMapping(descPrefix string, pm *portMapping) error {
	// Check that the client is running in the target node (otherwise UPnP usually reject the request)
	clientIP := lb.localAddress.String()
	if pm.nodeIP != "" && pm.nodeIP != clientIP {
		return fmt.Errorf("The local client (%s) cant be used to setup mappings to %s", clientIP, pm.nodeIP)
	}

	externalPort := uint16(pm.servicePort.Port)
	proto := string(pm.servicePort.Protocol)
	internalPort := uint16(pm.servicePort.NodePort)
	internalIP := pm.nodeIP
	desc := fmt.Sprintf("%s/%s", descPrefix, pm.servicePort.Name)
	lease := uint32(infinitePortMappingLeaseDuration)

	return lb.client.AddPortMapping("", externalPort, proto, internalPort, internalIP, true, desc, lease)
}

func (lb *LoadBalancer) deletePortMapping(pm *portMapping) error {
	externalPort := uint16(pm.servicePort.Port)
	proto := string(pm.servicePort.Protocol)
	return lb.client.DeletePortMapping("", externalPort, proto)
}

func getFirstNodeInternalIP(clientIP string, nodes []*k8s.Node) (string, error) {
	errCtx := "" // TODO: add ctx
	for _, node := range nodes {
		for _, address := range node.Status.Addresses {
			if address.Type == k8s.NodeInternalIP && address.Address == clientIP {
				return address.Address, nil
			}
		}
	}
	return "", fmt.Errorf("%s: no nodes with internal IP '%s'", errCtx, clientIP)
}

func getLocalAddressToHost(host string) net.IP {
	// To obtain the local address to a host, we create a socket (dial)
	// just to read the local address, that should have been chosen according to
	// local routing rules.
	// Both TCP and UDP should give the same IP address.
	// We use UDP since it is not connection oriented, so it in fact does not
	// send anything through the wire, unlike TCP handshake.
	// Also the port number is not used, but needed to do a proper call to dial.
	conn, err := net.Dial("udp", host+":12345")
	if err != nil {
		return nil
	}
	defer conn.Close()
	return net.ParseIP(strings.Split(conn.LocalAddr().String(), ":")[0])
}

func getExternalIP() (net.IP, error) {
	return externalip.DefaultConsensus(nil, nil).ExternalIP()
}

// Auxiliary functions

func fname() string {
	pc, _, _, _ := runtime.Caller(1)
	return runtime.FuncForPC(pc).Name()
}
