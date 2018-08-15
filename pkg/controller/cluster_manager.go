/*
Copyright 2015 The Kubernetes Authors.

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

package controller

import (
	"github.com/golang/glog"

	compute "google.golang.org/api/compute/v1"

	"k8s.io/ingress-gce/pkg/backends"
	"k8s.io/ingress-gce/pkg/context"
	"k8s.io/ingress-gce/pkg/healthchecks"
	"k8s.io/ingress-gce/pkg/instances"
	"k8s.io/ingress-gce/pkg/loadbalancers"
	"k8s.io/ingress-gce/pkg/utils"
)

// ClusterManager manages cluster resource pools.
type ClusterManager struct {
	ClusterNamer *utils.Namer
	instancePool instances.NodePool
	backendPool  backends.BackendPool
	l7Pool       loadbalancers.LoadBalancerPool

	// TODO: Refactor so we simply init a health check pool.
	healthChecker healthchecks.HealthChecker
}

// Init initializes the cluster manager.
func (c *ClusterManager) Init(zl instances.ZoneLister, pp backends.ProbeProvider) {
	c.instancePool.Init(zl)
	c.backendPool.Init(pp)
	// TODO: Initialize other members as needed.
}

func (c *ClusterManager) shutdown() error {
	if err := c.l7Pool.Shutdown(); err != nil {
		return err
	}
	// The backend pool will also delete instance groups.
	return c.backendPool.Shutdown()
}

func (c *ClusterManager) EnsureInstanceGroupsAndPorts(nodeNames []string, servicePorts []utils.ServicePort) ([]*compute.InstanceGroup, error) {
	// Convert to slice of NodePort int64s.
	ports := []int64{}
	for _, p := range uniq(servicePorts) {
		if !p.NEGEnabled {
			ports = append(ports, p.NodePort)
		}
	}

	// Create instance groups and set named ports.
	igs, err := c.instancePool.EnsureInstanceGroupsAndPorts(c.ClusterNamer.InstanceGroup(), ports)
	if err != nil {
		return nil, err
	}

	// Add/remove instances to the instance groups.
	if err = c.instancePool.Sync(nodeNames); err != nil {
		return nil, err
	}

	return igs, err
}

// GC garbage collects unused resources.
// - lbNames are the names of L7 loadbalancers we wish to exist. Those not in
//   this list are removed from the cloud.
// - nodePorts are the ports for which we want BackendServies. BackendServices
//   for ports not in this list are deleted.
// This method ignores googleapi 404 errors (StatusNotFound).
func (c *ClusterManager) GC(lbNames []string, nodePorts []utils.ServicePort) error {
	// On GC:
	// * Loadbalancers need to get deleted before backends.
	// * Backends are refcounted in a shared pool.
	// * We always want to GC backends even if there was an error in GCing
	//   loadbalancers, because the next Sync could rely on the GC for quota.
	// * There are at least 2 cases for backend GC:
	//   1. The loadbalancer has been deleted.
	//   2. An update to the url map drops the refcount of a backend. This can
	//      happen when an Ingress is updated, if we don't GC after the update
	//      we'll leak the backend.
	lbErr := c.l7Pool.GC(lbNames)
	beErr := c.backendPool.GC(nodePorts)
	if lbErr != nil {
		return lbErr
	}
	if beErr != nil {
		return beErr
	}

	// TODO(ingress#120): Move this to the backend pool so it mirrors creation
	if len(lbNames) == 0 {
		igName := c.ClusterNamer.InstanceGroup()
		glog.Infof("Deleting instance group %v", igName)
		if err := c.instancePool.DeleteInstanceGroup(igName); err != err {
			return err
		}
		glog.V(2).Infof("Shutting down firewall as there are no loadbalancers")
	}

	return nil
}

// NewClusterManager creates a cluster manager for shared resources.
// - namer: is the namer used to tag cluster wide shared resources.
// - defaultBackendNodePort: is the node port of glbc's default backend. This is
//	 the kubernetes Service that serves the 404 page if no urls match.
// - healthCheckPath: is the default path used for L7 health checks, eg: "/healthz".
// - defaultBackendHealthCheckPath: is the default path used for the default backend health checks.
func NewClusterManager(
	ctx *context.ControllerContext,
	namer *utils.Namer,
	healthCheckPath string,
	defaultBackendHealthCheckPath string) (*ClusterManager, error) {

	// Names are fundamental to the cluster, the uid allocator makes sure names don't collide.
	cluster := ClusterManager{ClusterNamer: namer}

	// NodePool stores GCE vms that are in this Kubernetes cluster.
	cluster.instancePool = instances.NewNodePool(ctx.Cloud, namer)

	// BackendPool creates GCE BackendServices and associated health checks.
	cluster.healthChecker = healthchecks.NewHealthChecker(ctx.Cloud, healthCheckPath, defaultBackendHealthCheckPath, cluster.ClusterNamer, ctx.DefaultBackendSvcPortID.Service)
	cluster.backendPool = backends.NewBackendPool(ctx.Cloud, ctx.Cloud, cluster.healthChecker, cluster.instancePool, cluster.ClusterNamer, ctx.BackendConfigEnabled, true)

	// L7 pool creates targetHTTPProxy, ForwardingRules, UrlMaps, StaticIPs.
	cluster.l7Pool = loadbalancers.NewLoadBalancerPool(ctx.Cloud, cluster.ClusterNamer)
	return &cluster, nil
}
