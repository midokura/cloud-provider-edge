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
	"fmt"
	"io"

	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog"
)

const (
	// ProviderName is the name of the edge provider
	ProviderName = "edge"
)

// Edge is an implementation of cloud provider Interface for edge deployments.
type Edge struct {
	LoadBalancerInstance *LoadBalancer
}

// init register the Edge Cloud Manager
func init() {
	cloudprovider.RegisterCloudProvider(ProviderName, func(config io.Reader) (cloudprovider.Interface, error) {
		cfg, err := ReadConfig(config)
		if err != nil {
			return nil, fmt.Errorf("unable to read edge cloud provider config file: %v", err)
		}
		edge, err := NewEdge(cfg)
		if err != nil {
			klog.V(1).Infof("New edge cloud provider client created failed with config")
		}
		return edge, err
	})
}

// NewEdge creates a new new instance of the Edge struct from a config struct
func NewEdge(cfg Config) (*Edge, error) {
	klog.Infof("New Edge Cloud Manager")
	cloud := Edge{}
	return &cloud, nil
}

// Initialize provides the cloud with a kubernetes client builder and may spawn goroutines
// to perform housekeeping or run custom controllers specific to the cloud provider.
// Any tasks started here should be cleaned up when the stop channel closes.
func (cloud Edge) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stop <-chan struct{}) {
}

// LoadBalancer returns a balancer interface, and true since the interface is supported.
func (cloud Edge) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	if cloud.LoadBalancerInstance == nil {
		loadBalancer, err := NewLoadBalancer()
		if err != nil {
			klog.Errorf("Error getting LoadBalancer interface: %v", err)
			return nil, false
		}
		cloud.LoadBalancerInstance = loadBalancer
	}
	klog.Infof("LoadBalancer API interface available")
	return cloud.LoadBalancerInstance, true
}

// Instances returns nil and false, since instances interface is not supported.
func (cloud Edge) Instances() (cloudprovider.Instances, bool) {
	return nil, false
}

// Zones returns nil and false, since zones interface is not supported.
func (cloud Edge) Zones() (cloudprovider.Zones, bool) {
	return nil, false
}

// Clusters returns nil and false, since clusters interface is not supported.
func (cloud Edge) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

// Routes returns nil and false, since routes interface is not supported.
func (cloud Edge) Routes() (cloudprovider.Routes, bool) {
	return nil, false
}

// ProviderName returns the cloud provider ID.
func (cloud Edge) ProviderName() string {
	return ProviderName
}

// HasClusterID returns true, as a ClusterID is required and set
func (cloud Edge) HasClusterID() bool {
	return true
}
