// Copyright 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validator

import (
	"fmt"
	"regexp"
	"strconv"

	kubeclient "k8s.io/client-go/kubernetes"

	"github.com/golang/glog"
	"k8s.io/api/extensions/v1beta1"
)

// validator is a concrete implementation of ValidatorInterface.
type validator struct {
}

var _ ValidatorInterface = &validator{}

// NewValidator returns a new Validator.
func NewValidator() ValidatorInterface {
	return &validator{}
}

// Validate performs pre-flight checks on the clusters and services.
func (v *validator) Validate(clients map[string]kubeclient.Interface, ing *v1beta1.Ingress) error {
	return serverVersionsNewEnough(clients)
}

// serverVersionsNewEnough returns an error if the version of any cluster is not supported.
func serverVersionsNewEnough(clients map[string]kubeclient.Interface) error {
	for key := range clients {
		glog.Infof("Checking client %s", key)
		discoveryClient := clients[key].Discovery()
		if discoveryClient == nil {
			return fmt.Errorf("no discovery client in %s client: %+v", key, clients[key])
		}
		ver, err := discoveryClient.ServerVersion()
		if err != nil {
			return fmt.Errorf("could not get discovery client to lookup server version: %s", err)
		}
		glog.Infof("ServerVersion: %+v", ver)
		major, minor, patch, err := parseVersion(ver.GitVersion)
		if err != nil {
			return err
		}
		if newEnough := serverVersionNewEnough(major, minor, patch); !newEnough {
			return fmt.Errorf("cluster %s (ver %d.%d.%d) is not running a supported kubernetes version. Need >= 1.8.1 and not 1.10.0",
				key, major, minor, patch)
		}

	}
	return nil
}

func serverVersionNewEnough(major, minor, patch uint64) bool {
	// 1.10.0 was buggy and doesn't work: kubernetes/ingress-gce#182.
	if major == 1 && minor == 10 && patch == 0 {
		return false
	}

	// 1.8.1 was the first supported release.
	if major > 1 {
		return true
	} else if major < 1 {
		return false
	}
	// Minor version
	if minor > 8 {
		return true
	} else if minor < 8 {
		return false
	}
	// Patch version
	if patch >= 1 {
		return true
	}
	return false
}

func parseVersion(version string) (uint64, uint64, uint64, error) {
	// Example string we're matching: v1.9.3-gke.0
	re := regexp.MustCompile("^v([0-9]*).([0-9]*).([0-9]*)")
	matches := re.FindStringSubmatch(version)
	glog.V(4).Infof("version string matches: %v\n", matches)

	if len(matches) < 3 {
		return 0, 0, 0, fmt.Errorf("did not find 3 components to version number %s", version)
	}
	// Major version
	major, err := strconv.ParseUint(matches[1], 10 /*base*/, 32 /*bitSize*/)
	if err != nil {
		return 0, 0, 0, err
	}
	// Minor version
	minor, err := strconv.ParseUint(matches[2], 10 /*base*/, 32 /*bitSize*/)
	if err != nil {
		return 0, 0, 0, err
	}
	// Patch version
	patch, err := strconv.ParseUint(matches[3], 10 /*bases*/, 32 /*bitSize*/)
	if err != nil {
		return 0, 0, 0, err
	}
	glog.V(2).Infof("Got version: major:", major, "minor:", minor, "patch:", patch)
	return major, minor, patch, nil
}
