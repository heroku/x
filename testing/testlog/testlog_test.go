package testlog

import "testing"

func TestNeedsQuoting(t *testing.T) {
	for _, s := range []string{"a", "A", "@", "-", ".", "_", "/", "@", "^", "+"} {
		if needsQuoting(s) {
			t.Errorf("%q shouldn't need quoting", s)
		}
	}
}
