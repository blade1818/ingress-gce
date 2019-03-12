/*
Copyright 2019 The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/testing"
	clientset "k8s.io/ingress-gce/pkg/backendconfig/client/clientset/versioned"
	cloudv1 "k8s.io/ingress-gce/pkg/backendconfig/client/clientset/versioned/typed/backendconfig/v1"
	fakecloudv1 "k8s.io/ingress-gce/pkg/backendconfig/client/clientset/versioned/typed/backendconfig/v1/fake"
	cloudv1beta1 "k8s.io/ingress-gce/pkg/backendconfig/client/clientset/versioned/typed/backendconfig/v1beta1"
	fakecloudv1beta1 "k8s.io/ingress-gce/pkg/backendconfig/client/clientset/versioned/typed/backendconfig/v1beta1/fake"
)

// NewSimpleClientset returns a clientset that will respond with the provided objects.
// It's backed by a very simple object tracker that processes creates, updates and deletions as-is,
// without applying any validations and/or defaults. It shouldn't be considered a replacement
// for a real clientset and is mostly useful in simple unit tests.
func NewSimpleClientset(objects ...runtime.Object) *Clientset {
	o := testing.NewObjectTracker(scheme, codecs.UniversalDecoder())
	for _, obj := range objects {
		if err := o.Add(obj); err != nil {
			panic(err)
		}
	}

	cs := &Clientset{}
	cs.discovery = &fakediscovery.FakeDiscovery{Fake: &cs.Fake}
	cs.AddReactor("*", "*", testing.ObjectReaction(o))
	cs.AddWatchReactor("*", func(action testing.Action) (handled bool, ret watch.Interface, err error) {
		gvr := action.GetResource()
		ns := action.GetNamespace()
		watch, err := o.Watch(gvr, ns)
		if err != nil {
			return false, nil, err
		}
		return true, watch, nil
	})

	return cs
}

// Clientset implements clientset.Interface. Meant to be embedded into a
// struct to get a default implementation. This makes faking out just the method
// you want to test easier.
type Clientset struct {
	testing.Fake
	discovery *fakediscovery.FakeDiscovery
}

func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	return c.discovery
}

var _ clientset.Interface = &Clientset{}

// CloudV1beta1 retrieves the CloudV1beta1Client
func (c *Clientset) CloudV1beta1() cloudv1beta1.CloudV1beta1Interface {
	return &fakecloudv1beta1.FakeCloudV1beta1{Fake: &c.Fake}
}

// CloudV1 retrieves the CloudV1Client
func (c *Clientset) CloudV1() cloudv1.CloudV1Interface {
	return &fakecloudv1.FakeCloudV1{Fake: &c.Fake}
}
