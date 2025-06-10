package testutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"
)

const (
	profilesDirName    = "../profiles"
	timestampFormat    = "20060102_150405"
	profilePermissions = 0755
	performanceLogTmpl = `
Test Duration: %s
Memory Allocations:
  Current memory in use: %d bytes
  Total memory allocated: %d bytes
  Allocation operations: %d allocs, %d frees (net: %d)
Heap Statistics:
  Heap allocation delta: %d bytes
  Heap objects delta: %d objects
Garbage Collection:
  GC cycles: %d
  GC CPU fraction: %.2f%%
Stack Statistics:
  Stack in use delta: %d bytes
  Stack system delta: %d bytes`
)

type TestProfile struct {
	TestName         string        `json:"test_name"`
	Timestamp        string        `json:"timestamp"`
	Duration         time.Duration `json:"duration"`
	AllocDelta       int64         `json:"alloc_delta"`
	TotalAllocDelta  int64         `json:"total_alloc_delta"`
	AllocsDelta      int64         `json:"allocs_delta"`
	FreesDelta       int64         `json:"frees_delta"`
	HeapAllocDelta   int64         `json:"heap_alloc_delta"`
	HeapObjectsDelta int64         `json:"heap_objects_delta"`
	GCCycles         int64         `json:"gc_cycles"`
	GCCPUFraction    float64       `json:"gc_cpu_fraction"`
	StackInuseDelta  int64         `json:"stack_inuse_delta"`
	StackSysDelta    int64         `json:"stack_sys_delta"`
}

type ProfileWriter struct {
	outputDir string
	testName  string
	t         *testing.T
}

func MeasureTestPerformance(t *testing.T) func() {
	t.Helper()
	var mStart runtime.MemStats
	runtime.ReadMemStats(&mStart)
	start := time.Now()
	testName := t.Name()

	return func() {
		var mEnd runtime.MemStats
		runtime.ReadMemStats(&mEnd)
		profile := createTestProfile(testName, start, mStart, mEnd)
		logTestPerformance(t, profile)

		writer := &ProfileWriter{
			outputDir: filepath.Join(profilesDirName, time.Now().Format(timestampFormat)),
			testName:  testName,
			t:         t,
		}
		writer.saveProfiles(profile)
	}
}

func createTestProfile(testName string, start time.Time, mStart, mEnd runtime.MemStats) TestProfile {
	return TestProfile{
		TestName:         testName,
		Timestamp:        time.Now().Format(time.RFC3339),
		Duration:         time.Since(start),
		AllocDelta:       int64(mEnd.Alloc - mStart.Alloc),
		TotalAllocDelta:  int64(mEnd.TotalAlloc - mStart.TotalAlloc),
		AllocsDelta:      int64(mEnd.Mallocs - mStart.Mallocs),
		FreesDelta:       int64(mEnd.Frees - mStart.Frees),
		HeapAllocDelta:   int64(mEnd.HeapAlloc - mStart.HeapAlloc),
		HeapObjectsDelta: int64(mEnd.HeapObjects - mStart.HeapObjects),
		GCCycles:         int64(mEnd.NumGC - mStart.NumGC),
		GCCPUFraction:    mEnd.GCCPUFraction,
		StackInuseDelta:  int64(mEnd.StackInuse - mStart.StackInuse),
		StackSysDelta:    int64(mEnd.StackSys - mStart.StackSys),
	}
}

func logTestPerformance(t *testing.T, profile TestProfile) {
	t.Logf(performanceLogTmpl,
		profile.Duration,
		profile.AllocDelta,
		profile.TotalAllocDelta,
		profile.AllocsDelta,
		profile.FreesDelta,
		profile.AllocsDelta-profile.FreesDelta,
		profile.HeapAllocDelta,
		profile.HeapObjectsDelta,
		profile.GCCycles,
		profile.GCCPUFraction*100,
		profile.StackInuseDelta,
		profile.StackSysDelta,
	)
}

func (pw *ProfileWriter) saveProfiles(profile TestProfile) {
	if err := os.MkdirAll(pw.outputDir, profilePermissions); err != nil {
		pw.t.Logf("Failed to create profile directory: %v", err)
		return
	}

	pw.saveJSONProfile(profile)
	pw.saveHeapProfile()
	pw.saveGoroutineProfile()
	pw.saveAllocsProfile()
	pw.saveCPUProfile()
	pw.saveMemoryProfile()
}

func (pw *ProfileWriter) saveJSONProfile(profile TestProfile) {
	file := pw.createProfileFile(pw.testName + ".json")
	if file == nil {
		return
	}
	defer file.Close()
	if err := json.NewEncoder(file).Encode(profile); err != nil {
		pw.t.Logf("Failed to encode profile: %v", err)
	}
}

func (pw *ProfileWriter) createProfileFile(name string) *os.File {
	file, err := os.Create(filepath.Join(pw.outputDir, name))
	if err != nil {
		pw.t.Logf("Failed to create profile file %s: %v", name, err)
		return nil
	}
	return file
}

func (pw *ProfileWriter) saveHeapProfile() {
	file := pw.createProfileFile(fmt.Sprintf("%s_heap.prof", pw.testName))
	if file == nil {
		return
	}
	defer file.Close()
	if err := pprof.WriteHeapProfile(file); err != nil {
		pw.t.Logf("Failed to write heap profile: %v", err)
	}
}

func (pw *ProfileWriter) saveGoroutineProfile() {
	file := pw.createProfileFile(fmt.Sprintf("%s_goroutines.prof", pw.testName))
	if file == nil {
		return
	}
	defer file.Close()
	if err := pprof.Lookup("goroutine").WriteTo(file, 0); err != nil {
		pw.t.Logf("Failed to write goroutine profile: %v", err)
	}
}

func (pw *ProfileWriter) saveAllocsProfile() {
	file := pw.createProfileFile(fmt.Sprintf("%s_allocs.prof", pw.testName))
	if file == nil {
		return
	}
	defer file.Close()
	if err := pprof.Lookup("allocs").WriteTo(file, 0); err != nil {
		pw.t.Logf("Failed to write allocs profile: %v", err)
	}
}

func (pw *ProfileWriter) saveCPUProfile() {
	file := pw.createProfileFile("cpu.prof")
	if file == nil {
		return
	}
	if err := pprof.StartCPUProfile(file); err != nil {
		pw.t.Logf("Failed to start CPU profile: %v", err)
		file.Close()
		return
	}

	pprof.StopCPUProfile()
	if err := file.Close(); err != nil {
		pw.t.Logf("Failed to close CPU profile: %v", err)
	}
}

func (pw *ProfileWriter) saveMemoryProfile() {
	file := pw.createProfileFile("mem.prof")
	if file == nil {
		return
	}
	defer file.Close()
	if err := pprof.WriteHeapProfile(file); err != nil {
		pw.t.Logf("Failed to write memory profile: %v", err)
	}
}
