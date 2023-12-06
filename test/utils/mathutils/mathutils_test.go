package mathutils

import "testing"

func TestNearestMultiple(t *testing.T) {
	res := NearestMultiple(10, 15)
	if res != 20 {
		t.Errorf("Expected result to be 20, got %d", res)
	}
	res = NearestMultiple(15, 10)
	if res != 15 {
		t.Errorf("Multiple (%d) was > input (%d). Expected result to be %d, got %d", 15, 10, 15, res)
	}
}
