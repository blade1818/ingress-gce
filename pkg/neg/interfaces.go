/*
Copyright 2017 The Kubernetes Authors.

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

package neg

import (
	computebeta "google.golang.org/api/compute/v0.beta"
)

// MetworkEndpointGroupCloud is an interface for managing gce network endpoint group.
type NetworkEndpointGroupCloud interface {
	GetNetworkEndpointGroup(name string, zone string) (*computebeta.NetworkEndpointGroup, error)
	ListNetworkEndpointGroup(zone string) ([]*computebeta.NetworkEndpointGroup, error)
	AggregatedListNetworkEndpointGroup() (map[string][]*computebeta.NetworkEndpointGroup, error)
	CreateNetworkEndpointGroup(neg *computebeta.NetworkEndpointGroup, zone string) error
	DeleteNetworkEndpointGroup(name string, zone string) error
	AttachNetworkEndpoints(name, zone string, endpoints []*computebeta.NetworkEndpoint) error
	DetachNetworkEndpoints(name, zone string, endpoints []*computebeta.NetworkEndpoint) error
	ListNetworkEndpoints(name, zone string, showHealthStatus bool) ([]*computebeta.NetworkEndpointWithHealthStatus, error)
	NetworkURL() string
	SubnetworkURL() string
}

// networkEndpointGroupNamer is an interface for generating network endpoint group name.
type networkEndpointGroupNamer interface {
	NEG(namespace, name string, port int32) string
	IsNEG(name string) bool
}

// zoneGetter is an interface for retrieve zone related information
type zoneGetter interface {
	ListZones() ([]string, error)
	GetZoneForNode(name string) (string, error)
}

// negSyncer is an interface to interact with syncer
type negSyncer interface {
	// Start starts the syncer. This call is synchronous. It will return after syncer is started.
	Start() error
	// Stop stops the syncer. This call is asynchronous. It will not block until syncer is stopped.
	Stop()
	// Sync signals the syncer to sync NEG. This call is asynchronous. Syncer will sync once it becomes idle.
	Sync() bool
	// IsStopped returns true if syncer is stopped
	IsStopped() bool
	// IsShuttingDown returns true if syncer is shutting down
	IsShuttingDown() bool
}

// negSyncerManager is an interface for controllers to manage syncer
type negSyncerManager interface {
	// EnsureSyncer ensures corresponding syncers are started and stops any unnecessary syncer
	// portMap is a map of ServicePort Port to TargetPort
	EnsureSyncers(namespace, name string, portMap PortNameMap) error
	// StopSyncer stops all syncers related to the service. This call is asynchronous. It will not wait for all syncers to stop.
	StopSyncer(namespace, name string)
	// Sync signals all syncers related to the service to sync. This call is asynchronous.
	Sync(namespace, name string)
	// GC garbage collects network endpoint group and syncers
	GC() error
	// ShutDown shuts down the manager
	ShutDown()
}
