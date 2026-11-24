package actions

import (
	"testing"

	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
)

func TestMgmtConditionTrue(t *testing.T) {
	c := &management.Cluster{Conditions: []management.ClusterCondition{
		{Type: "AgentDeployed", Status: "True"},
		{Type: "Connected", Status: "False"},
	}}

	for _, tc := range []struct {
		typ  string
		want bool
	}{{"AgentDeployed", true}, {"Connected", false}, {"Missing", false}} {
		if got := mgmtConditionTrue(c, tc.typ); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.typ, got, tc.want)
		}
	}
}
