package subcommands

import "testing"

func TestRancherImageRepository(t *testing.T) {
	for version, want := range map[string]string{
		"v2.16.0": "rancher/rancher-server", "2.16.0": "rancher/rancher-server",
		"2.16.0-alpha1": "rancher/rancher-server", "v2.16.0-rc2": "rancher/rancher-server",
		"3.0.0": "rancher/rancher-server", "2.15.3": "rancher/rancher", "v2.15.0": "rancher/rancher",
		"2.16-head": "rancher/rancher-server", "2.15.9": "rancher/rancher",
		"2.17.0": "rancher/rancher-server", "10.0.0": "rancher/rancher-server",
		"": "rancher/rancher", "2": "rancher/rancher", "garbage": "rancher/rancher",
	} {
		if got := rancherImageRepository(version); got != want {
			t.Errorf("%q: got %q, want %q", version, got, want)
		}
	}
}

func TestRancherMajorMinor(t *testing.T) {
	for version, want := range map[string][2]int{
		"2.14.3": {2, 14}, "2.15.0": {2, 15}, "v2.15.1-rc2": {2, 15}, "2.16-head": {2, 16}, "": {0, 0}, "garbage": {0, 0},
	} {
		if major, minor := rancherMajorMinor(version); major != want[0] || minor != want[1] {
			t.Errorf("%q: got %d.%d, want %d.%d", version, major, minor, want[0], want[1])
		}
	}
}

func TestGetRancherValsJSONImageRepository(t *testing.T) {
	for _, tc := range []struct{ version, override, want string }{
		{"2.16.0", "", "rancher/rancher-server"},
		{"2.16-head", "", "rancher/rancher-server"},
		{"2.15.0", "", ""},
		{"2.16.0", "custom/rancher", ""},
	} {
		vals := getRancherValsJSON(tc.version, tc.override, "v"+tc.version, "pw", "host", nil, 1)
		image := vals["image"].(map[string]any)
		got, _ := image["repository"].(string)

		if got != tc.want {
			t.Errorf("%+v: repository %q", tc, got)
		}

		if tc.override != "" && vals["rancherImage"] != tc.override {
			t.Errorf("%+v: rancherImage %v", tc, vals["rancherImage"])
		}

		if tc.override == "" {
			if _, ok := vals["rancherImage"]; ok {
				t.Errorf("%+v: unexpected rancherImage", tc)
			}
		}

		if tc.version == "2.15.0" && (len(image) != 1 || image["tag"] == nil) {
			t.Errorf("%+v: image should only contain tag: %v", tc, image)
		}
	}
}
