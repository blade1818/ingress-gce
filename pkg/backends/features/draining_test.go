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

package features

import (
	"testing"

	v1beta1 "k8s.io/ingress-gce/pkg/apis/cloud/v1beta1"
	"k8s.io/ingress-gce/pkg/composite"
	"k8s.io/ingress-gce/pkg/utils"
)

func TestEnsureDraining(t *testing.T) {
	testCases := []struct {
		desc           string
		sp             utils.ServicePort
		be             *composite.BackendService
		updateExpected bool
	}{
		{
			desc: "connection draining timeout setting is defined on serviceport but missing from spec, update needed",
			sp: utils.ServicePort{
				BackendConfig: &v1beta1.BackendConfig{
					Spec: v1beta1.BackendConfigSpec{
						ConnectionDraining: &v1beta1.ConnectionDrainingConfig{
							DrainingTimeoutSec: 111,
						},
					},
				},
			},
			be: &composite.BackendService{
				ConnectionDraining: &composite.ConnectionDraining{},
			},
			updateExpected: true,
		},
		{
			desc: "connection draining setting are missing from spec, no update needed",
			sp: utils.ServicePort{
				BackendConfig: &v1beta1.BackendConfig{
					Spec: v1beta1.BackendConfigSpec{},
				},
			},
			be: &composite.BackendService{
				ConnectionDraining: &composite.ConnectionDraining{
					DrainingTimeoutSec: 111,
				},
			},
			updateExpected: false,
		},
		{
			desc: "connection draining settings are identical, no update needed",
			sp: utils.ServicePort{
				BackendConfig: &v1beta1.BackendConfig{
					Spec: v1beta1.BackendConfigSpec{
						ConnectionDraining: &v1beta1.ConnectionDrainingConfig{
							DrainingTimeoutSec: 111,
						},
					},
				},
			},
			be: &composite.BackendService{
				ConnectionDraining: &composite.ConnectionDraining{
					DrainingTimeoutSec: 111,
				},
			},
			updateExpected: false,
		},
		{
			desc: "connection draining settings differs, update needed",
			sp: utils.ServicePort{
				BackendConfig: &v1beta1.BackendConfig{
					Spec: v1beta1.BackendConfigSpec{
						ConnectionDraining: &v1beta1.ConnectionDrainingConfig{
							DrainingTimeoutSec: 111,
						},
					},
				},
			},
			be: &composite.BackendService{
				ConnectionDraining: &composite.ConnectionDraining{
					DrainingTimeoutSec: 222,
				},
			},
			updateExpected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := EnsureDraining(tc.sp, tc.be)
			if result != tc.updateExpected {
				t.Errorf("%v: expected %v but got %v", tc.desc, tc.updateExpected, result)
			}
		})
	}
}
