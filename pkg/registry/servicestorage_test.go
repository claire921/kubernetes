/*
Copyright 2014 Google Inc. All rights reserved.

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

package registry

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/apiserver"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/cloudprovider"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
)

func TestServiceRegistry(t *testing.T) {
	memory := MakeMemoryRegistry()
	fakeCloud := &cloudprovider.FakeCloud{}
	machines := []string{"foo", "bar", "baz"}

	storage := MakeServiceRegistryStorage(memory, fakeCloud, MakeMinionRegistry(machines))

	svc := api.Service{
		JSONBase: api.JSONBase{ID: "foo"},
		Selector: map[string]string{"bar": "baz"},
	}
	c, _ := storage.Create(svc)
	<-c

	if len(fakeCloud.Calls) != 0 {
		t.Errorf("Unexpected call(s): %#v", fakeCloud.Calls)
	}
	srv, err := memory.GetService(svc.ID)
	expectNoError(t, err)
	if srv == nil {
		t.Errorf("Failed to find service: %s", svc.ID)
	}
}

func TestServiceStorageValidatesCreate(t *testing.T) {
	memory := MakeMemoryRegistry()
	storage := MakeServiceRegistryStorage(memory, nil, nil)

	failureCases := map[string]api.Service{
		"empty ID": {
			JSONBase: api.JSONBase{ID: ""},
			Selector: map[string]string{"bar": "baz"},
		},
		"empty selector": {
			JSONBase: api.JSONBase{ID: "foo"},
			Selector: map[string]string{},
		},
	}
	for _, failureCase := range failureCases {
		c, err := storage.Create(failureCase)
		if c != nil {
			t.Errorf("Expected nil channel")
		}
		if err == nil {
			t.Errorf("Expected to get an error")
		}
	}
}

func TestServiceStorageValidatesUpdate(t *testing.T) {
	memory := MakeMemoryRegistry()
	memory.CreateService(api.Service{
		JSONBase: api.JSONBase{ID: "foo"},
		Selector: map[string]string{"bar": "baz"},
	})
	storage := MakeServiceRegistryStorage(memory, nil, nil)

	failureCases := map[string]api.Service{
		"empty ID": {
			JSONBase: api.JSONBase{ID: ""},
			Selector: map[string]string{"bar": "baz"},
		},
		"empty selector": {
			JSONBase: api.JSONBase{ID: "foo"},
			Selector: map[string]string{},
		},
	}
	for _, failureCase := range failureCases {
		c, err := storage.Update(failureCase)
		if c != nil {
			t.Errorf("Expected nil channel")
		}
		if err == nil {
			t.Errorf("Expected to get an error")
		}
	}
}

func TestServiceRegistryExternalService(t *testing.T) {
	memory := MakeMemoryRegistry()
	fakeCloud := &cloudprovider.FakeCloud{}
	machines := []string{"foo", "bar", "baz"}

	storage := MakeServiceRegistryStorage(memory, fakeCloud, MakeMinionRegistry(machines))

	svc := api.Service{
		JSONBase:                   api.JSONBase{ID: "foo"},
		Selector:                   map[string]string{"bar": "baz"},
		CreateExternalLoadBalancer: true,
	}
	c, _ := storage.Create(svc)
	<-c

	if len(fakeCloud.Calls) != 2 || fakeCloud.Calls[0] != "get-zone" || fakeCloud.Calls[1] != "create" {
		t.Errorf("Unexpected call(s): %#v", fakeCloud.Calls)
	}
	srv, err := memory.GetService(svc.ID)
	expectNoError(t, err)
	if srv == nil {
		t.Errorf("Failed to find service: %s", svc.ID)
	}
}

func TestServiceRegistryExternalServiceError(t *testing.T) {
	memory := MakeMemoryRegistry()
	fakeCloud := &cloudprovider.FakeCloud{
		Err: fmt.Errorf("test error"),
	}
	machines := []string{"foo", "bar", "baz"}

	storage := MakeServiceRegistryStorage(memory, fakeCloud, MakeMinionRegistry(machines))

	svc := api.Service{
		JSONBase:                   api.JSONBase{ID: "foo"},
		Selector:                   map[string]string{"bar": "baz"},
		CreateExternalLoadBalancer: true,
	}
	c, _ := storage.Create(svc)
	<-c

	if len(fakeCloud.Calls) != 1 || fakeCloud.Calls[0] != "get-zone" {
		t.Errorf("Unexpected call(s): %#v", fakeCloud.Calls)
	}
	srv, err := memory.GetService("foo")
	if !apiserver.IsNotFound(err) {
		if err != nil {
			t.Errorf("memory.GetService(%q) failed with %v; expected failure with not found error", svc.ID, err)
		} else {
			t.Errorf("memory.GetService(%q) = %v; expected failure with not found error", svc.ID, srv)
		}
	}
}

func TestServiceRegistryDelete(t *testing.T) {
	memory := MakeMemoryRegistry()
	fakeCloud := &cloudprovider.FakeCloud{}
	machines := []string{"foo", "bar", "baz"}

	storage := MakeServiceRegistryStorage(memory, fakeCloud, MakeMinionRegistry(machines))

	svc := api.Service{
		JSONBase: api.JSONBase{ID: "foo"},
		Selector: map[string]string{"bar": "baz"},
	}
	memory.CreateService(svc)

	c, _ := storage.Delete(svc.ID)
	<-c

	if len(fakeCloud.Calls) != 0 {
		t.Errorf("Unexpected call(s): %#v", fakeCloud.Calls)
	}
	srv, err := memory.GetService(svc.ID)
	if !apiserver.IsNotFound(err) {
		if err != nil {
			t.Errorf("memory.GetService(%q) failed with %v; expected failure with not found error", svc.ID, err)
		} else {
			t.Errorf("memory.GetService(%q) = %v; expected failure with not found error", svc.ID, srv)
		}
	}
}

func TestServiceRegistryDeleteExternal(t *testing.T) {
	memory := MakeMemoryRegistry()
	fakeCloud := &cloudprovider.FakeCloud{}
	machines := []string{"foo", "bar", "baz"}

	storage := MakeServiceRegistryStorage(memory, fakeCloud, MakeMinionRegistry(machines))

	svc := api.Service{
		JSONBase:                   api.JSONBase{ID: "foo"},
		Selector:                   map[string]string{"bar": "baz"},
		CreateExternalLoadBalancer: true,
	}
	memory.CreateService(svc)

	c, _ := storage.Delete(svc.ID)
	<-c

	if len(fakeCloud.Calls) != 1 || fakeCloud.Calls[0] != "delete" {
		t.Errorf("Unexpected call(s): %#v", fakeCloud.Calls)
	}
	srv, err := memory.GetService(svc.ID)
	if !apiserver.IsNotFound(err) {
		if err != nil {
			t.Errorf("memory.GetService(%q) failed with %v; expected failure with not found error", svc.ID, err)
		} else {
			t.Errorf("memory.GetService(%q) = %v; expected failure with not found error", svc.ID, srv)
		}
	}
}

func TestServiceRegistryMakeLinkVariables(t *testing.T) {
	service := api.Service{
		JSONBase:      api.JSONBase{ID: "foo"},
		Selector:      map[string]string{"bar": "baz"},
		ContainerPort: util.IntOrString{Kind: util.IntstrString, StrVal: "a-b-c"},
	}
	vars := makeLinkVariables(service, "mars")
	for _, v := range vars {
		if !util.IsCIdentifier(v.Name) {
			t.Errorf("Environment variable name is not valid: %v", v.Name)
		}
	}
}
