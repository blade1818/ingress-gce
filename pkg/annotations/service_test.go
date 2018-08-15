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

package annotations

import (
	"reflect"
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/ingress-gce/pkg/flags"
)

func TestNEGService(t *testing.T) {
	for _, tc := range []struct {
		svc     *v1.Service
		neg     bool
		ingress bool
		exposed bool
	}{
		{
			svc: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						NEGAnnotationKey: `{"ingress":true}`,
					},
				},
			},
			neg:     true,
			ingress: true,
			exposed: false,
		},
		{
			svc: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						NEGAnnotationKey: `{"exposed_ports":{"80":{}}}`,
					},
				},
			},
			neg:     true,
			ingress: false,
			exposed: true,
		},
		{
			svc: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						NEGAnnotationKey: `{"ingress":true,"exposed_ports":{"80":{}}}`,
					},
				},
			},
			neg:     true,
			ingress: true,
			exposed: true,
		},
		{
			svc:     &v1.Service{},
			neg:     false,
			ingress: false,
			exposed: false,
		},
	} {
		svc := FromService(tc.svc)
		if neg := svc.NEGEnabled(); neg != tc.neg {
			t.Errorf("for service %+v; svc.NEGEnabled() = %v; want %v", tc.svc, neg, tc.neg)
		}

		if ing := svc.NEGEnabledForIngress(); ing != tc.ingress {
			t.Errorf("for service %+v; svc.NEGEnabledForIngress() = %v; want %v", tc.svc, ing, tc.ingress)
		}

		if exposed := svc.NEGExposed(); exposed != tc.exposed {
			t.Errorf("for service %+v; svc.NEGExposed() = %v; want %v", tc.svc, exposed, tc.exposed)
		}
	}
}

func TestService(t *testing.T) {
	for _, tc := range []struct {
		svc             *v1.Service
		appProtocolsErr bool
		appProtocols    map[string]AppProtocol
		http2           bool
	}{
		{
			svc:          &v1.Service{},
			appProtocols: map[string]AppProtocol{},
		},
		{
			svc: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						GoogleServiceApplicationProtocolKey: `{"80": "HTTP", "443": "HTTPS"}`,
					},
				},
			},
			appProtocols: map[string]AppProtocol{"80": "HTTP", "443": "HTTPS"},
		},
		{
			svc: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						GoogleServiceApplicationProtocolKey: `{"80": "HTTP", "443": "HTTPS"}`,
						ServiceApplicationProtocolKey:       `{"81": "HTTP", "444": "HTTPS"}`,
					},
				},
			},
			appProtocols: map[string]AppProtocol{"81": "HTTP", "444": "HTTPS"},
		},
		{
			svc: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ServiceApplicationProtocolKey: `{"80": "HTTP", "443": "HTTPS"}`,
					},
				},
			},
			appProtocols: map[string]AppProtocol{"80": "HTTP", "443": "HTTPS"},
		},
		{
			svc: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ServiceApplicationProtocolKey: `{"443": "HTTP2"}`,
					},
				},
			},
			appProtocols:    map[string]AppProtocol{"443": "HTTP2"},
			appProtocolsErr: true, // Without the http2 flag enabled, expect error
		},
		{
			svc: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ServiceApplicationProtocolKey: `{"443": "HTTP2"}`,
					},
				},
			},
			appProtocols: map[string]AppProtocol{"443": "HTTP2"},
			http2:        true,
		},
		{
			svc: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ServiceApplicationProtocolKey: `invalid`,
					},
				},
			},
			appProtocolsErr: true,
		},
		{
			svc: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ServiceApplicationProtocolKey: `{"SSH": "22"}`,
					},
				},
			},
			appProtocolsErr: true,
		},
	} {
		flags.F.Features.Http2 = tc.http2
		svc := FromService(tc.svc)
		ap, err := svc.ApplicationProtocols()
		if tc.appProtocolsErr {
			if err == nil {
				t.Errorf("for service %+v; svc.ApplicationProtocols() = _, %v; want _, error", tc.svc, err)
			}
			continue
		}
		if err != nil || !reflect.DeepEqual(ap, tc.appProtocols) {
			t.Errorf("for service %+v; svc.ApplicationProtocols() = %v, %v; want %v, nil", tc.svc, ap, err, tc.appProtocols)
		}
	}
}

func TestBackendConfigs(t *testing.T) {
	testcases := []struct {
		desc            string
		svc             *v1.Service
		expectedConfigs *BackendConfigs
		expectedErr     error
	}{
		{
			desc:        "no backendConfig annotation",
			svc:         &v1.Service{},
			expectedErr: ErrBackendConfigAnnotationMissing,
		},
		{
			desc: "single backendConfig",
			svc: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						BackendConfigKey: `{"ports":{"http": "config-http"}}`,
					},
				},
			},
			expectedConfigs: &BackendConfigs{
				Ports: map[string]string{
					"http": "config-http",
				},
			},
		},
		{
			desc: "multiple backendConfigs",
			svc: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						BackendConfigKey: `{"ports":{"http": "config-http", "https": "config-https"}}`,
					},
				},
			},
			expectedConfigs: &BackendConfigs{
				Ports: map[string]string{
					"http":  "config-http",
					"https": "config-https",
				},
			},
		}, {
			desc: "multiple backendConfigs with default",
			svc: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						BackendConfigKey: `{"default": "config-default", "ports":{"http": "config-http", "https": "config-https"}}`,
					},
				},
			},
			expectedConfigs: &BackendConfigs{
				Default: "config-default",
				Ports: map[string]string{
					"http":  "config-http",
					"https": "config-https",
				},
			},
		},
		{
			desc: "invalid backendConfig annotation",
			svc: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						BackendConfigKey: `invalid`,
					},
				},
			},
			expectedErr: ErrBackendConfigInvalidJSON,
		},
		{
			desc: "wrong field name in backendConfig annotation",
			svc: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						BackendConfigKey: `{"portstypo":{"https": "config-https"}}`,
					},
				},
			},
			expectedErr: ErrBackendConfigNoneFound,
		},
	}

	for _, tc := range testcases {
		svc := FromService(tc.svc)
		configs, err := svc.GetBackendConfigs()
		if !reflect.DeepEqual(configs, tc.expectedConfigs) || tc.expectedErr != err {
			t.Errorf("%s: for annotations %+v; svc.GetBackendConfigs() = %v, %v; want %v, %v", tc.desc, svc.v, configs, err, tc.expectedConfigs, tc.expectedErr)
		}
	}
}

func TestNegAnnotation(t *testing.T) {
	testcases := []struct {
		desc        string
		annotation  string
		expected    NegAnnotation
		expectedErr error
	}{
		{
			desc:        "no expose NEG annotation",
			annotation:  "",
			expectedErr: ErrExposeNegAnnotationMissing,
		},
		{
			desc:        "invalid expose NEG annotation",
			annotation:  "invalid",
			expectedErr: ErrExposeNegAnnotationInvalid,
		},
		{
			desc: "NEG annotation references existing service ports",
			expected: NegAnnotation{
				ExposedPorts: map[int32]NegAttributes{80: NegAttributes{}, 443: NegAttributes{}},
			},
			annotation: `{"exposed_ports":{"80":{},"443":{}}}`,
		},

		{
			desc:       "NEGServicePort takes the union of known ports and ports referenced in the annotation",
			annotation: `{"exposed_ports":{"80":{}}}`,
			expected: NegAnnotation{
				ExposedPorts: map[int32]NegAttributes{80: NegAttributes{}},
			},
		},
	}

	for _, tc := range testcases {
		service := &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{},
			},
		}

		t.Run(tc.desc, func(t *testing.T) {
			if len(tc.annotation) > 0 {
				service.Annotations[NEGAnnotationKey] = tc.annotation
			}

			svc := FromService(service)
			exposeNegStruct, err := svc.NegAnnotation()

			if tc.expectedErr == nil && err != nil {
				t.Errorf("ExpectedNEGServicePorts to not return an error, got: %v", err)
			}

			if !reflect.DeepEqual(exposeNegStruct, tc.expected) {
				t.Errorf("Expected NEGServicePorts to equal: %v; got: %v", tc.expected, exposeNegStruct.ExposedPorts)
			}

			if tc.expectedErr != nil && err != tc.expectedErr {
				t.Errorf("Expected NEGServicePorts to return a %v error, got: %v", tc.expectedErr, err)
			}
		})
	}
}
