package uid

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestNewString(t *testing.T) {
	v := NewKSUIDString()
	if len(v) != 27 {
		t.Fatal("Expected 27 characters, got", len(v))
	}
}

func TestNewBytes(t *testing.T) {
	v := NewKSUIDBytes()
	if len(v) != 20 {
		t.Fatal("Expected 20 bytes, got", len(v))
	}
}

func TestConstructionTimestamp(t *testing.T) {
	x := NewKSUID()
	nowTime := time.Now().Round(1 * time.Minute)
	xTime := x.Time().Round(1 * time.Minute)

	if xTime != nowTime {
		t.Fatal(xTime, "!=", nowTime)
	}
}

func TestNil(t *testing.T) {
	if !NilKSUID.IsNil() {
		t.Fatal("Nil should be Nil!")
	}

	x, _ := FromKSUIDBytes(make([]byte, byteLength))
	if !x.IsNil() {
		t.Fatal("Zero-byte array should be Nil!")
	}
}

func TestEncoding(t *testing.T) {
	x, _ := FromKSUIDBytes(make([]byte, byteLength))
	if !x.IsNil() {
		t.Fatal("Zero-byte array should be Nil!")
	}

	encoded := x.String()
	expected := strings.Repeat("0", stringEncodedLength)

	if encoded != expected {
		t.Fatal("expected", expected, "encoded", encoded)
	}
}

func TestPadding(t *testing.T) {
	b := make([]byte, byteLength)
	for i := 0; i < byteLength; i++ {
		b[i] = 255
	}

	x, _ := FromKSUIDBytes(b)
	xEncoded := x.String()
	nilEncoded := NilKSUID.String()

	if len(xEncoded) != len(nilEncoded) {
		t.Fatal("Encoding should produce equal-length strings for zero and max case")
	}
}

func TestParse(t *testing.T) {
	_, err := ParseKSUID("123")
	if !errors.Is(err, errStrSize) {
		t.Fatal("Expected Parsing a 3-char string to return an error")
	}

	parsed, err := ParseKSUID(strings.Repeat("0", stringEncodedLength))
	if err != nil {
		t.Fatal("Unexpected error", err)
	}

	if Compare(parsed, NilKSUID) != 0 {
		t.Fatal("Parsing all-zeroes string should equal Nil value",
			"expected:", NilKSUID,
			"actual:", parsed)
	}

	maxBytes := make([]byte, byteLength)
	for i := 0; i < byteLength; i++ {
		maxBytes[i] = 255
	}
	maxBytesKSUID, err := FromKSUIDBytes(maxBytes)
	if err != nil {
		t.Fatal("Unexpected error", err)
	}

	maxParseKSUID, err := ParseKSUID(maxStringEncoded)
	if err != nil {
		t.Fatal("Unexpected error", err)
	}

	if Compare(maxBytesKSUID, maxParseKSUID) != 0 {
		t.Fatal("String decoder broke for max string")
	}
}

func TestIssue25(t *testing.T) {
	// https://github.com/inovacc/ksuid/issues/25
	for _, s := range []string{
		"aaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"aWgEPTl1tmebfsQzFP4bxwgy80!",
	} {
		_, err := ParseKSUID(s)
		if !errors.Is(err, errStrValue) {
			t.Error("invalid KSUID representations cannot be successfully parsed, got err =", err)
		}
	}
}

func TestEncodeAndDecode(t *testing.T) {
	x := NewKSUID()
	builtFromEncodedString, err := ParseKSUID(x.String())
	if err != nil {
		t.Fatal("Unexpected error", err)
	}

	if Compare(x, builtFromEncodedString) != 0 {
		t.Fatal("Parse(X).String() != X")
	}
}

func TestMarshalText(t *testing.T) {
	var id1 = NewKSUID()
	var id2 KSUID

	if err := id2.UnmarshalText([]byte(id1.String())); err != nil {
		t.Fatal(err)
	}

	if id1 != id2 {
		t.Fatal(id1, "!=", id2)
	}

	if b, err := id2.MarshalText(); err != nil {
		t.Fatal(err)
	} else if s := string(b); s != id1.String() {
		t.Fatal(s)
	}
}

func TestMarshalBinary(t *testing.T) {
	var id1 = NewKSUID()
	var id2 KSUID

	if err := id2.UnmarshalBinary(id1.Bytes()); err != nil {
		t.Fatal(err)
	}

	if id1 != id2 {
		t.Fatal(id1, "!=", id2)
	}

	if b, err := id2.MarshalBinary(); err != nil {
		t.Fatal(err)
	} else if bytes.Compare(b, id1.Bytes()) != 0 {
		t.Fatal("bad binary form:", id2)
	}
}

func TestMashalJSON(t *testing.T) {
	var id1 = NewKSUID()
	var id2 KSUID

	if b, err := json.Marshal(id1); err != nil {
		t.Fatal(err)
	} else if err := json.Unmarshal(b, &id2); err != nil {
		t.Fatal(err)
	} else if id1 != id2 {
		t.Error(id1, "!=", id2)
	}
}

func TestFlag(t *testing.T) {
	var id1 = NewKSUID()
	var id2 KSUID

	fset := flag.NewFlagSet("test", flag.ContinueOnError)
	fset.Var(&id2, "id", "the KSUID")

	if err := fset.Parse([]string{"-id", id1.String()}); err != nil {
		t.Fatal(err)
	}

	if id1 != id2 {
		t.Error(id1, "!=", id2)
	}
}

func TestSqlValuer(t *testing.T) {
	id, _ := ParseKSUID(maxStringEncoded)

	if v, err := id.Value(); err != nil {
		t.Error(err)
	} else if s, ok := v.(string); !ok {
		t.Error("not a string value")
	} else if s != maxStringEncoded {
		t.Error("bad string value::", s)
	}
}

func TestSqlValuerNilValue(t *testing.T) {
	if v, err := NilKSUID.Value(); err != nil {
		t.Error(err)
	} else if v != nil {
		t.Errorf("bad nil value: %v", v)
	}
}

func TestSqlScanner(t *testing.T) {
	id1 := NewKSUID()
	id2 := NewKSUID()

	tests := []struct {
		id    KSUID
		value any
	}{
		{NilKSUID, nil},
		{id1, id1.String()},
		{id2, id2.Bytes()},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%T", test.value), func(t *testing.T) {
			var id KSUID

			if err := id.Scan(test.value); err != nil {
				t.Error(err)
			}

			if id != test.id {
				t.Error("bad KSUID:")
				t.Logf("expected %v", test.id)
				t.Logf("found    %v", id)
			}
		})
	}
}

func TestAppend(t *testing.T) {
	for _, repr := range []string{"0pN1Own7255s7jwpwy495bAZeEa", "aWgEPTl1tmebfsQzFP4bxwgy80V"} {
		k, _ := ParseKSUID(repr)
		a := make([]byte, 0, stringEncodedLength)

		a = append(a, "?: "...)
		a = k.Append(a)

		if s := string(a); s != "?: "+repr {
			t.Error(s)
		}
	}
}

func TestSort(t *testing.T) {
	ids1 := [11]KSUID{}
	ids2 := [11]KSUID{}

	for i := range ids1 {
		ids1[i] = NewKSUID()
	}

	ids2 = ids1
	sort.Slice(ids2[:], func(i, j int) bool {
		return Compare(ids2[i], ids2[j]) < 0
	})

	Sort(ids1[:])

	if !IsSorted(ids1[:]) {
		t.Error("not sorted")
	}

	if ids1 != ids2 {
		t.Error("bad order:")
		t.Log(ids1)
		t.Log(ids2)
	}
}

func TestPrevNext(t *testing.T) {
	tests := []struct {
		id   KSUID
		prev KSUID
		next KSUID
	}{
		{
			id:   NilKSUID,
			prev: MaxKSUID,
			next: KSUID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		},
		{
			id:   MaxKSUID,
			prev: KSUID{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe},
			next: NilKSUID,
		},
	}

	for _, test := range tests {
		t.Run(test.id.String(), func(t *testing.T) {
			testPrevNext(t, test.id, test.prev, test.next)
		})
	}
}

func TestGetTimestamp(t *testing.T) {
	nowTime := time.Now()
	x, _ := NewRandomWithTime(nowTime)
	xTime := int64(x.Timestamp())
	unix := nowTime.Unix()
	if xTime != unix-epochStamp {
		t.Fatal(xTime, "!=", unix)
	}
}

func testPrevNext(t *testing.T, id, prev, next KSUID) {
	id1 := id.Prev()
	id2 := id.Next()

	if id1 != prev {
		t.Error("previous id of the nil KSUID is wrong:", id1, "!=", prev)
	}

	if id2 != next {
		t.Error("next id of the nil KSUID is wrong:", id2, "!=", next)
	}
}

func BenchmarkAppend(b *testing.B) {
	a := make([]byte, 0, stringEncodedLength)
	k := NewKSUID()

	for i := 0; i != b.N; i++ {
		k.Append(a)
	}
}

func BenchmarkString(b *testing.B) {
	k := NewKSUID()

	for i := 0; i != b.N; i++ {
		_ = k.String()
	}
}

func BenchmarkParseKSUID(b *testing.B) {
	for i := 0; i != b.N; i++ {
		_, _ = ParseKSUID(maxStringEncoded)
	}
}

func BenchmarkCompare(b *testing.B) {
	k1 := NewKSUID()
	k2 := NewKSUID()

	for i := 0; i != b.N; i++ {
		Compare(k1, k2)
	}
}

func BenchmarkSort(b *testing.B) {
	ids1 := [101]KSUID{}
	ids2 := [101]KSUID{}

	for i := range ids1 {
		ids1[i] = NewKSUID()
	}

	for i := 0; i != b.N; i++ {
		ids2 = ids1
		Sort(ids2[:])
	}
}

func BenchmarkNewKSUID(b *testing.B) {
	b.Run("with crypto rand", func(b *testing.B) {
		SetRandKSUID(nil)
		for i := 0; i != b.N; i++ {
			NewKSUID()
		}
	})
	b.Run("with math rand", func(b *testing.B) {
		SetRandKSUID(FastRander)
		for i := 0; i != b.N; i++ {
			NewKSUID()
		}
	})
}
