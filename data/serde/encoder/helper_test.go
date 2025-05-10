package encoder

import (
	"testing"

	"github.com/inovacc/utils/v2/random/random"
)

func TestWrapString(t *testing.T) {
	str := random.RandomString(500000)
	wrapped := wrapString(str, 80)
	unwrap := unwrapString(wrapped)

	if unwrap != str {
		t.Errorf("Expected %s, got %s", str, unwrap)
	}
}
