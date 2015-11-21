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
	"testing"

	rktapi "github.com/coreos/rkt/api/v1alpha"
	"github.com/stretchr/testify/assert"
)

func TestCheckVersion(t *testing.T) {
	fr := newFakeRktInterface()
	fs := newFakeSystemd()
	r := &Runtime{apisvc: fr, systemd: fs}

	fr.info = rktapi.Info{
		RktVersion:  "1.2.3+git",
		AppcVersion: "1.2.4+git",
		ApiVersion:  "1.2.6-alpha",
	}
	fs.version = "100"
	tests := []struct {
		minimumRktBinVersion   string
		recommendRktBinVersion string
		minimumAppcVersion     string
		minimumRktApiVersion   string
		minimumSystemdVersion  string
		err                    error
	}{
		// Good versions.
		{
			"1.2.3",
			"1.2.3",
			"1.2.4",
			"1.2.5",
			"99",
			nil,
		},
		// Good versions.
		{
			"1.2.3+git",
			"1.2.3+git",
			"1.2.4+git",
			"1.2.6-alpha",
			"100",
			nil,
		},
		// Requires greater binary version.
		{
			"1.2.4",
			"1.2.4",
			"1.2.4",
			"1.2.6-alpha",
			"100",
			fmt.Errorf("rkt: binary version is too old(%v), requires at least %v", fr.info.RktVersion, "1.2.4"),
		},
		// Requires greater Appc version.
		{
			"1.2.3",
			"1.2.3",
			"1.2.5",
			"1.2.6-alpha",
			"100",
			fmt.Errorf("rkt: Appc version is too old(%v), requires at least %v", fr.info.AppcVersion, "1.2.5"),
		},
		// Requires greater API version.
		{
			"1.2.3",
			"1.2.3",
			"1.2.4",
			"1.2.6",
			"100",
			fmt.Errorf("rkt: API version is too old(%v), requires at least %v", fr.info.ApiVersion, "1.2.6"),
		},
		// Requires greater API version.
		{
			"1.2.3",
			"1.2.3",
			"1.2.4",
			"1.2.7",
			"100",
			fmt.Errorf("rkt: API version is too old(%v), requires at least %v", fr.info.ApiVersion, "1.2.7"),
		},
		// Requires greater systemd version.
		{
			"1.2.3",
			"1.2.3",
			"1.2.4",
			"1.2.7",
			"101",
			fmt.Errorf("rkt: systemd version(%v) is too old, requires at least %v", fs.version, "101"),
		},
	}

	for i, tt := range tests {
		testCaseHint := fmt.Sprintf("test case #%d", i)
		err := r.checkVersion(tt.minimumRktBinVersion, tt.recommendRktBinVersion, tt.minimumAppcVersion, tt.minimumRktApiVersion, tt.minimumSystemdVersion)
		assert.Equal(t, err, tt.err, testCaseHint)

		if err == nil {
			assert.Equal(t, r.binVersion.String(), fr.info.RktVersion, testCaseHint)
			assert.Equal(t, r.appcVersion.String(), fr.info.AppcVersion, testCaseHint)
			assert.Equal(t, r.apiVersion.String(), fr.info.ApiVersion, testCaseHint)
		}
	}
}

func TestListImages(t *testing.T) {
	fr := newFakeRktInterface()
	fs := newFakeSystemd()
	r := &Runtime{apisvc: fr, systemd: fs}

	tests := []struct {
		images []*rktapi.Image
	}{
		{},
		{
			[]*rktapi.Image{
				&rktapi.Image{
					Id:      "sha512-a2fb8f390702",
					Name:    "quay.io/coreos/alpine-sh",
					Version: "latest",
				},
			},
		},
		{
			[]*rktapi.Image{
				&rktapi.Image{
					Id:      "sha512-a2fb8f390702",
					Name:    "quay.io/coreos/alpine-sh",
					Version: "latest",
				},
				&rktapi.Image{
					Id:      "sha512-c6b597f42816",
					Name:    "coreos.com/rkt/stage1-coreos",
					Version: "0.10.0",
				},
			},
		},
	}

	for _, tt := range tests {
		fr.images = tt.images

		//testCaseHint := fmt.Sprintf("test case #%d", i)
		images, err := r.ListImages()
		if err != nil {
			t.Errorf("%v", err)
		}
		if len(tt.images) != len(images) {
			t.Errorf("incorrect number of images returned, expecting=%d, got=%d", len(tt.images), len(images))
		}
		for i, image := range images {
			if tt.images[i].Id != image.ID {
				t.Errorf("mismatched image IDs, expecting=%s, got=%s", tt.images[i].Id, image.ID)
			}
			if len(image.Tags) != 1 || tt.images[i].Name != image.Tags[0] {
				t.Errorf("mismatched image tags, expecting=%v, got=%v", []string{tt.images[i].Name}, image.Tags)
			}
		}
	}
}
