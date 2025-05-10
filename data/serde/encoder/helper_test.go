package encoder

import (
	"testing"

	"github.com/inovacc/toolkit/data/algorithm/random"
)

func TestWrapString(t *testing.T) {
	str := random.RandomString(500000)
	wrapped := wrapString(str, 80)
	unwrap := unwrapString(wrapped)

	if unwrap != str {
		t.Errorf("Expected %s, got %s", str, unwrap)
	}
}
