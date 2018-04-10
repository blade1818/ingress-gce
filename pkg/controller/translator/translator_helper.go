package translator

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/pkg/apis/extensions"
	"k8s.io/client-go/tools/cache"

	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getK8sServicesForIngressBackend returns a list of k8s services which
// correspond to the ServicePort for the IngressBackend.
func getK8sServicesForIngressBackend(ib extensions.IngressBackend, namespace string, svcListers []cache.Indexer) ([]api_v1.Service, error) {
	svcs := make([]api_v1.Service, 0)
	for _, l := range svcListers {
		obj, exists, err := l.Get(
			&api_v1.Service{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      ib.ServiceName,
					Namespace: namespace,
				},
			})
		if !exists {
			return nil, fmt.Errorf("service %v/%v not found in store", namespace, ib.ServiceName)
		}
		if err != nil {
			return nil, err
		}
		svc := obj.(*api_v1.Service)
		svcs = append(svcs, svc)
	}
	return svcs, nil
}

// getNodePortForIngressBackend returns the NodePort for the Service referenced in
// the IngressBackend.
func getNodePortForIngressBackend(ib extensions.IngressBackend, svc api_v1.Service) (int64, error) {
	var svcPort *api_v1.ServicePort
	// Find the ServicePort which matches the ServicePort specified in IngressBackend.
	for _, sp := range svc.Spec.Ports {
		spCopy := sp
		switch ib.ServicePort.Type {
		case intstr.Int:
			if sp.Port == ib.ServicePort.IntVal {
				svcPort = &spCopy
				break
			}
		default:
			if sp.Name == ib.ServicePort.StrVal {
				svcPort = &spCopy
				break
			}
		}
	}

	if svcPort == nil {
		return -1, fmt.Errorf("could not find matching nodeport from service")
	}

	return int64(svcPort.NodePort), nil
}
