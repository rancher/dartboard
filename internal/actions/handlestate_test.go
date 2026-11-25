package actions

import (
	"reflect"
	"testing"
)

func TestProvisionedRancherNames(t *testing.T) {
	got := ProvisionedRancherNames(map[string]*ClusterStatus{
		"a-0-1": {Provisioned: true, RancherName: "vsphere-zz"},
		"a-0-0": {Provisioned: true, RancherName: "vsphere-aa"},
		"old":   {Provisioned: true},
		"half":  {RancherName: "vsphere-half"},
		"cust":  {Registered: true},
	})
	want := []string{"vsphere-aa", "vsphere-zz"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
