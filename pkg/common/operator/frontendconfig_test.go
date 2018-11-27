package operator

import (
	"testing"

	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/ingress-gce/pkg/test"
)

func TestDoesIngressReferenceFrontendConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		desc     string
		ing      *extensions.Ingress
		expected bool
	}{
		{
			desc:     "ingress with no frontend config annotation",
			ing:      test.IngressWithoutFrontendConfig,
			expected: false,
		},
		{
			desc:     "ingress with different frontend config",
			ing:      test.IngressWithOtherFrontendConfig,
			expected: false,
		},
		{
			desc:     "ingress in different namspace",
			ing:      test.IngressWithFrontendConfigOtherNamespace,
			expected: false,
		},
		{
			desc:     "ingress with expected frontend config",
			ing:      test.IngressWithFrontendConfig,
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := doesIngressReferenceFrontendConfig(tc.ing, test.FrontendConfig)
			if result != tc.expected {
				t.Fatalf("Expected result to be %v, got %v", tc.expected, result)
			}
		})
	}
}
