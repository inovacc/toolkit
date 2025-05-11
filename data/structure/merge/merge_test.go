package structure

import (
	"reflect"
	"testing"
	"time"
)

type sampleConfig struct {
	Name       string
	Enabled    bool
	MaxSize    int
	Timeout    time.Duration
	Nested     *string
	SliceField []string
}

func TestMergeZeroFields(t *testing.T) {
	defaultVal := "default"
	base := &sampleConfig{
		Name:       "Existing",
		Enabled:    true,
		MaxSize:    100,
		Timeout:    10 * time.Second,
		Nested:     &defaultVal,
		SliceField: []string{"keep"},
	}

	override := &sampleConfig{
		Name:       "",
		Enabled:    false,
		MaxSize:    0,
		Timeout:    0,
		Nested:     nil,
		SliceField: nil,
	}

	MergeZeroFields(override, base)

	if !reflect.DeepEqual(override, base) {
		t.Errorf("Expected merged config to match base, got: %+v", override)
	}
}

func TestCopyExportedFields(t *testing.T) {
	src := &sampleConfig{
		Name:    "Copied",
		Enabled: true,
		MaxSize: 200,
		Timeout: 5 * time.Second,
	}
	dst := &sampleConfig{}

	if err := CopyExportedFields(dst, src); err != nil {
		t.Errorf("error copying exported fields: %v", err)
	}

	if !reflect.DeepEqual(dst, src) {
		t.Errorf("Expected dst to match src, got: %+v", dst)
	}
}

func TestZeroStruct(t *testing.T) {
	defaultVal := "non-nil"
	cfg := &sampleConfig{
		Name:       "Test",
		Enabled:    true,
		MaxSize:    10,
		Timeout:    time.Second,
		Nested:     &defaultVal,
		SliceField: []string{"one", "two"},
	}

	ZeroStruct(cfg)

	if cfg.Name != "" || cfg.Enabled || cfg.MaxSize != 0 || cfg.Timeout != 0 || cfg.Nested != nil || cfg.SliceField != nil {
		t.Errorf("Expected all fields to be zeroed, got: %+v", cfg)
	}
}

func TestStructToMap(t *testing.T) {
	val := "hello"
	cfg := sampleConfig{
		Name:       "MapMe",
		Enabled:    true,
		MaxSize:    42,
		Nested:     &val,
		SliceField: []string{"x"},
	}

	result := StructToMap(cfg)

	if result["Name"] != "MapMe" || result["Enabled"] != true || result["MaxSize"] != 42 {
		t.Errorf("Unexpected result in StructToMap: %+v", result)
	}
}
