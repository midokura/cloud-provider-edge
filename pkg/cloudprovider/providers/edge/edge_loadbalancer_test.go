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
	"net"
	"reflect"
	"testing"

	"k8s.io/api/core/v1"
)

type mapping struct {
	proto        string
	externalPort uint16
	internalIP   string
	internalPort uint16
}

type mockClient struct {
	t              *testing.T
	removed, added []mapping
}

func (client *mockClient) AddPortMapping(host string, externalPort uint16, proto string, internalPort uint16, internalIP string, enabled bool, desc string, lease uint32) (err error) {
	mapping := mapping{
		proto:        proto,
		externalPort: externalPort,
		internalIP:   internalIP,
		internalPort: internalPort,
	}
	client.added = append(client.added, mapping)
	return nil
}

func (client *mockClient) DeletePortMapping(host string, externalPort uint16, proto string) (err error) {
	if client.added != nil {
		client.t.Errorf("unexpected AddPortMapping before DeletePortMapping")
	}
	mapping := mapping{
		proto:        proto,
		externalPort: externalPort,
	}
	client.removed = append(client.removed, mapping)
	return nil
}

func newMockClient(t *testing.T) *mockClient {
	return &mockClient{t: t}
}

func TestPatchLoadBalancer(t *testing.T) {
	nodeIP := "192.0.2.1"
	mockClient := newMockClient(t)
	lb := LoadBalancer{
		client:       mockClient,
		localAddress: net.ParseIP(nodeIP),
	}
	oldMapping := []portMapping{
		{ // This will be removed
			servicePort: v1.ServicePort{
				Name:     "svc-port-1",
				Protocol: "TCP",
				Port:     12345,
				NodePort: 34567,
			},
			nodeIP: nodeIP,
		},
		{ // This will stay
			servicePort: v1.ServicePort{
				Name:     "svc-port-2",
				Protocol: "TCP",
				Port:     12345,
				NodePort: 23456,
			},
			nodeIP: nodeIP,
		},
	}
	newMapping := []portMapping{
		{ // Unchanged
			servicePort: v1.ServicePort{
				Name:     "svc-port-2",
				Protocol: "TCP",
				Port:     12345,
				NodePort: 23456,
			},
			nodeIP: nodeIP,
		},
		{ // This will be added
			servicePort: v1.ServicePort{
				Name:     "svc-port-1",
				Protocol: "TCP",
				Port:     12345,
				NodePort: 23456, // Changed
			},
			nodeIP: nodeIP,
		},
	}
	err := lb.patchLoadBalancer("foo", oldMapping, newMapping)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	actualRemoved := mockClient.removed
	expectedRemoved := []mapping{
		{
			proto:        "TCP",
			externalPort: 12345,
		},
	}
	if !reflect.DeepEqual(actualRemoved, expectedRemoved) {
		t.Errorf("got %v\nwant %v", actualRemoved, expectedRemoved)
	}
	actualAdded := mockClient.added
	expectedAdded := []mapping{
		{
			proto:        "TCP",
			externalPort: 12345,
			internalIP:   nodeIP,
			internalPort: 23456,
		},
	}
	if !reflect.DeepEqual(actualAdded, expectedAdded) {
		t.Errorf("got %v\nwant %v", actualAdded, expectedAdded)
	}
}
