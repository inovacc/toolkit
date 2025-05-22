package uid

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestSequence(t *testing.T) {
	seq := Sequence{Seed: NewKSUID()}

	if minValue, maxValue := seq.Bounds(); minValue == maxValue {
		t.Error("minValue and maxValue of KSUID range must differ when no ids have been generated")
	}

	for i := 0; i <= math.MaxUint16; i++ {
		id, err := seq.Next()
		if err != nil {
			t.Fatal(err)
		}
		if j := int(binary.BigEndian.Uint16(id[len(id)-2:])); j != i {
			t.Fatalf("expected %d but got %d in %s", i, j, id)
		}
	}

	if _, err := seq.Next(); err == nil {
		t.Fatal("no error returned after exhausting the id generator")
	}

	if minValue, maxValue := seq.Bounds(); minValue != maxValue {
		t.Error("after all KSUIDs were generated the minValue and maxValue must be equal")
	}
}
