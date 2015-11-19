/*
Copyright 2015 The Kubernetes Authors All rights reserved.

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

package rkt

import (
	"fmt"

	"github.com/coreos/go-semver/semver"
	rktapi "github.com/coreos/rkt/api/v1alpha"
	"github.com/golang/glog"
	"golang.org/x/net/context"
)

// rktVersion implementes kubecontainer.Version interface by implementing
// Compare() and String() (which is implemented by the underlying semver.Version)
type rktVersion struct {
	*semver.Version
}

func (r rktVersion) Compare(other string) (int, error) {
	v, err := semver.NewVersion(other)
	if err != nil {
		return -1, err
	}

	if r.LessThan(*v) {
		return -1, nil
	}
	if v.LessThan(*r.Version) {
		return 1, nil
	}
	return 0, nil
}

// On success, compareVersion returns:
// -1 if v1 is less than v2.
// 1 if v1 is greater than v2.
// 0 if v1 equals v2.
func compareVersion(v1, v2 string) (int, error) {
	semv, err := semver.NewVersion(v1)
	if err != nil {
		return -1, err
	}
	rktv := rktVersion{semv}
	return rktv.Compare(v2)
}

// checkVersion tests whether the rkt/systemd/rkt-api-service that meet the version requirement.
// If all version requirements are met, it returns nil.
func (r *Runtime) checkVersion(minimumRktBinVersion, recommendRktBinVersion, minimumAppcVersion, minimumRktApiVersion, minimumSystemdVersion string) error {
	// Check systemd version.
	systemdVersion, err := r.systemd.Version()
	if err != nil {
		return err
	}
	result, err := systemdVersion.Compare(minimumSystemdVersion)
	if err != nil {
		return err
	}
	if result < 0 {
		return fmt.Errorf("rkt: systemd version(%v) is too old, requires at least %v", systemdVersion, minimumSystemdVersion)
	}

	// Example for the version strings returned by GetInfo():
	// RktVersion:"0.10.0+gitb7349b1" AppcVersion:"0.7.1" ApiVersion:"1.0.0-alpha"
	resp, err := r.apisvc.GetInfo(context.Background(), &rktapi.GetInfoRequest{})
	if err != nil {
		return err
	}

	// Check rkt binary version.
	result, err = compareVersion(resp.Info.RktVersion, minimumRktBinVersion)
	if err != nil {
		return err
	}
	if result < 0 {
		return fmt.Errorf("rkt: binary version is too old(%v), requires at least %v", resp.Info.RktVersion, minimumRktBinVersion)
	}
	result, err = compareVersion(resp.Info.RktVersion, recommendRktBinVersion)
	if err != nil {
		return err
	}
	if result != 0 {
		// TODO(yifan): Record an event to expose the information.
		glog.Warningf("rkt: current binary version %q is not recommended (recommended version %q)", resp.Info.RktVersion, recommendRktBinVersion)
	}

	// Check Appc version.
	result, err = compareVersion(resp.Info.AppcVersion, minimumAppcVersion)
	if err != nil {
		return err
	}
	if result < 0 {
		return fmt.Errorf("rkt: Appc version is too old(%v), requires at least %v", resp.Info.AppcVersion, minimumAppcVersion)
	}

	// Check rkt API version.
	result, err = compareVersion(resp.Info.ApiVersion, minimumRktApiVersion)
	if err != nil {
		return err
	}
	if result < 0 {
		return fmt.Errorf("rkt: API version is too old(%v), requires at least %v", resp.Info.ApiVersion, minimumRktApiVersion)
	}

	v, _ := semver.NewVersion(resp.Info.AppcVersion)
	r.appcVersion = rktVersion{v}

	v, _ = semver.NewVersion(resp.Info.RktVersion)
	r.binVersion = rktVersion{v}

	v, _ = semver.NewVersion(resp.Info.ApiVersion)
	r.apiVersion = rktVersion{v}

	r.systemdVersion = systemdVersion
	return nil
}
