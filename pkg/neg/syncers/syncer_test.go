/*
Copyright 2018 The Kubernetes Authors.

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

package syncers

import (
	"testing"
	"time"

	"fmt"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/record"
	"k8s.io/ingress-gce/pkg/context"
	"k8s.io/ingress-gce/pkg/utils"
)

type syncerTester struct {
	syncer
	// keep track of the number of syncs
	syncCount int
	// syncError is true, then sync function return error
	syncError bool
	// blockSync is true, then sync function is blocked on channel
	blockSync bool
	ch        chan interface{}
}

// sync sleeps for 3 seconds
func (t *syncerTester) sync() error {
	t.syncCount += 1
	if t.syncError {
		return fmt.Errorf("sync error")
	}
	if t.blockSync {
		<-t.ch
	}
	return nil
}

func newTestNegSyncer() *syncerTester {
	testNegName := "test-neg-name"
	kubeClient := fake.NewSimpleClientset()
	namer := utils.NewNamer(clusterID, "")
	ctxConfig := context.ControllerContextConfig{
		NEGEnabled:              true,
		Namespace:               apiv1.NamespaceAll,
		ResyncPeriod:            1 * time.Second,
		DefaultBackendSvcPortID: defaultBackend,
	}
	context := context.NewControllerContext(kubeClient, nil, nil, nil, nil, namer, ctxConfig)
	negSyncerKey := NegSyncerKey{
		Namespace:  testServiceNamespace,
		Name:       testServiceName,
		Port:       80,
		TargetPort: "80",
	}

	s := &syncerTester{
		syncer: *newSyncer(
			negSyncerKey,
			testNegName,
			context.ServiceInformer.GetIndexer(),
			record.NewFakeRecorder(100),
		),
		syncCount: 0,
		blockSync: false,
		syncError: false,
		ch:        make(chan interface{}),
	}
	s.SetSyncFunc(s.sync)
	return s
}

func TestStartAndStopNoopSyncer(t *testing.T) {
	syncer := newTestNegSyncer()
	if !syncer.IsStopped() {
		t.Fatalf("Syncer is not stopped after creation.")
	}
	if syncer.IsShuttingDown() {
		t.Fatalf("Syncer is shutting down after creation.")
	}

	if err := syncer.Start(); err != nil {
		t.Fatalf("Failed to start syncer: %v", err)
	}
	if syncer.IsStopped() {
		t.Fatalf("Syncer is stopped after Start.")
	}
	if syncer.IsShuttingDown() {
		t.Fatalf("Syncer is shutting down after Start.")
	}

	// blocks sync function
	syncer.blockSync = true
	syncer.Stop()
	if !syncer.IsShuttingDown() {
		// assume syncer needs 5 second for sync
		t.Fatalf("Syncer is not shutting down after Start.")
	}

	if !syncer.IsStopped() {
		t.Fatalf("Syncer is not stopped after Stop.")
	}

	// unblock sync function
	syncer.ch <- struct{}{}
	if err := wait.PollImmediate(time.Second, 3*time.Second, func() (bool, error) {
		return !syncer.IsShuttingDown() && syncer.IsStopped(), nil
	}); err != nil {
		t.Fatalf("Syncer failed to shutdown: %v", err)
	}

	if err := syncer.Start(); err != nil {
		t.Fatalf("Failed to restart syncer: %v", err)
	}
	if syncer.IsStopped() {
		t.Fatalf("Syncer is stopped after restart.")
	}
	if syncer.IsShuttingDown() {
		t.Fatalf("Syncer is shutting down after restart.")
	}

	syncer.Stop()
	if !syncer.IsStopped() {
		t.Fatalf("Syncer is not stopped after Stop.")
	}
}

func TestRetryOnSyncError(t *testing.T) {
	maxRetry := 3
	syncer := newTestNegSyncer()
	syncer.syncError = true
	if err := syncer.Start(); err != nil {
		t.Fatalf("Failed to start syncer: %v", err)
	}
	syncer.backoff = NewExponentialBackendOffHandler(maxRetry, 0, 0)

	if err := wait.PollImmediate(time.Second, 5*time.Second, func() (bool, error) {
		// In 5 seconds, syncer should be able to retry 3 times.
		return syncer.syncCount == maxRetry+1, nil
	}); err != nil {
		t.Errorf("Syncer failed to retry and record error: %v", err)
	}

	if syncer.syncCount != maxRetry+1 {
		t.Errorf("Expect sync count to be %v, but got %v", maxRetry+1, syncer.syncCount)
	}
}
