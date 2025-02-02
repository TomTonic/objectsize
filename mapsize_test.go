package objectsize

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"
)

func TestSizeOfMap(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected uint64
	}{
		{
			name:     "Empty map",
			input:    map[string]int32{},
			expected: 56,
		},
		{
			name:     "Map with one element",
			input:    map[string]int32{"key1": 1},
			expected: 88, // base (56 bytes) + 1 bucket (8 bytes) + 1 key/value pair (20+4=24 bytes)
		},
		{
			name:     "Map with two element2",
			input:    map[string]int32{"key1": 1, "key2": 2},
			expected: 112, // base (56 bytes) + 1 bucket (8 bytes) + 2 key/value pairs (20+4=24 bytes x2)
		},
		{
			name:     "Map with three elements",
			input:    map[string]int32{"key1": 1, "key2": 2, "key3": 3},
			expected: 136, // base (56 bytes) + 1 bucket (8 bytes) + 3 key/value pairs (20+4=24 bytes x3)
		},
		{
			name:     "Map with four elements",
			input:    map[string]int32{"key1": 1, "key2": 2, "key3": 3, "key4": 4},
			expected: 160, // base (56 bytes) + 1 bucket (8 bytes) + 4 key/value pairs (20+4=24 bytes x4)
		},
		{
			name:     "Map with nine elements",
			input:    map[string]int32{"key0": 0, "key1": 1, "key2": 2, "key3": 3, "key4": 4, "key5": 5, "key6": 6, "key7": 7, "key8": 8},
			expected: 288, // base (56 bytes) + 2 bucket (8 bytes x2) + 9 key/value pairs (20+4=24 bytes x9)
		},
		{
			name:     "Map with int32/int32 types",
			input:    map[int32]int32{1: 1, 2: 2, 3: 3},
			expected: 88, // base (56 bytes) + 1 bucket (8 bytes) + 3 key/value pairs (4+4=8 bytes x3)
		},
		{
			name:     "Map with 17 int32/int32 types",
			input:    map[int32]int32{1: 1, 2: 2, 3: 3, 4: 4, 5: 5, 6: 6, 7: 7, 8: 8, 9: 9, 10: 10, 11: 11, 12: 12, 13: 13, 14: 14, 15: 15, 16: 16, 17: 17},
			expected: 224, // base (56 bytes) + 4 buckets (8 bytes x4) + 17 key/value pairs (4+4=8 bytes x17)
		},
		{
			name:     "Map with int64/int64 types",
			input:    map[int64]int64{1: 1, 2: 2, 3: 3},
			expected: 112, // base (56 bytes) + 1 bucket (8 bytes) + 3 key/value pairs (8+8=16 bytes x3)
		},
		{
			name:     "Map with nested maps",
			input:    map[string]map[string]int64{"outr": {"innr": 1}},
			expected: 176, // (base (56 bytes) + 1 bucket (8 bytes) + 1 key/value pair (20+8=28 bytes)) x2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Of(tt.input)
			if err != nil {
				t.Errorf("sizeOf() error = %v", err)
				return
			}
			if got != tt.expected {
				t.Errorf("sizeOf() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func measureMemMap3232(numberOfMaps int, numberOfEntries int) float64 {
	mapsArray := make([]map[int64]int64, numberOfMaps)

	var startMem, endMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	for i := range numberOfMaps {
		mapsArray[i] = make(map[int64]int64, numberOfEntries)
		for j := range numberOfEntries {
			mapsArray[i][rand.Int63()] = int64(j)
		}
	}
	//time.Sleep(5 * time.Second)

	runtime.GC()
	runtime.ReadMemStats(&endMem)
	//usedMem := (endMem.HeapAlloc + endMem.StackInuse + endMem.StackSys) - (startMem.HeapAlloc + startMem.StackInuse + startMem.StackSys)
	usedMem := endMem.HeapAlloc - startMem.HeapAlloc
	result := float64(usedMem) / float64(numberOfMaps)

	// make something stupid with the data to make sure it has not jet been deleted
	somevar := uint64(0)
	for i := range numberOfMaps {
		for j := range numberOfEntries {
			somevar += uint64(mapsArray[i][int64(j)])
		}
	}
	fmt.Println(somevar)
	return result
}

func checkSingleInstance(numberOfEntries int) (uint64, error) {
	m := make(map[int64]int64)
	for j := range numberOfEntries {
		m[rand.Int63()] = int64(j)
	}
	got, err := Of(m)
	return got, err
}

func TestSizeOfMapManytimes(t *testing.T) {
	tests := []struct {
		name            string
		numberOfMaps    int
		numberOfEntries int
	}{
		{
			name:            "1000000 small maps",
			numberOfMaps:    1000,
			numberOfEntries: 57,
		},
		{
			name:            "1000000 small maps",
			numberOfMaps:    1000,
			numberOfEntries: 58,
		},
		{
			name:            "1000000 small maps",
			numberOfMaps:    1000,
			numberOfEntries: 59,
		},
		{
			name:            "1000000 small maps",
			numberOfMaps:    1000,
			numberOfEntries: 60,
		},
		{
			name:            "1000000 small maps",
			numberOfMaps:    1000,
			numberOfEntries: 61,
		},
		{
			name:            "1000000 small maps",
			numberOfMaps:    1000,
			numberOfEntries: 62,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := measureMemMap3232(tt.numberOfMaps, tt.numberOfEntries)
			got, err := checkSingleInstance(tt.numberOfEntries)
			if err != nil {
				t.Errorf("sizeOf() error = %v", err)
				return
			}
			if float64(got) != float64(expected) {
				t.Errorf("sizeOf() = %v, want %v", got, expected)
			}
		})
	}
}
