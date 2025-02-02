package objectsize

import (
	"math/rand"
	"reflect"
	"runtime"
	"testing"
)

const (
	// Constants for 32-bit architecture
	expectedMapBaseSize32 = 4 + // hmap.count (int)
		1 + // hmap.flags
		1 + // hmap.B
		2 + // hmap.noverflow
		4 + // hmap.hash0
		4 + // hmap.buckets (unsafe.Pointer)
		4 + // hmap.oldbuckets (unsafe.Pointer)
		4 + // hmap.nevacuate (uintptr)
		4 // hmap.extra (*mapextra)

	// Constants for 64-bit architecture
	expectedMapBaseSize64 = 8 + // hmap.count (int)
		1 + // hmap.flags
		1 + // hmap.B
		2 + // hmap.noverflow
		4 + // hmap.hash0
		8 + // hmap.buckets (unsafe.Pointer)
		8 + // hmap.oldbuckets (unsafe.Pointer)
		8 + // hmap.nevacuate (uintptr)
		8 // hmap.extra (*mapextra)
)

func TestCalcMapBaseSize(t *testing.T) {
	var expectedSize uint64
	switch runtime.GOARCH {
	case "386", "arm", "mips", "mipsle", "wasm":
		expectedSize = expectedMapBaseSize32
	case "amd64", "arm64", "ppc64", "ppc64le", "mips64", "mips64le", "s390x":
		expectedSize = expectedMapBaseSize64
	default:
		t.Fatalf("Unsupported architecture: %s", runtime.GOARCH)
	}

	actualSize := calcMapBaseSize()

	if actualSize != expectedSize {
		t.Errorf("Expected %d, but got %d", expectedSize, actualSize)
	}
}

func TestCompareCalcMapBaseSizeWithMeasurement(t *testing.T) {
	numberOfMaps := 1000000
	tolerance := 0.005
	mapsArray := make([]map[string]int32, numberOfMaps)

	var startMem, endMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	for i := range numberOfMaps {
		mapsArray[i] = map[string]int32{}
	}

	runtime.GC()
	runtime.ReadMemStats(&endMem)
	//usedMem := (endMem.HeapAlloc + endMem.StackInuse + endMem.StackSys) - (startMem.HeapAlloc + startMem.StackInuse + startMem.StackSys)
	usedMem := endMem.HeapAlloc - startMem.HeapAlloc
	measuredMemPerMap := float64(usedMem) / float64(numberOfMaps)

	// do something useless with the data to make sure it has not yet been garbage collected
	zerosum := int64(0)
	for i := range numberOfMaps {
		zerosum += int64(len(mapsArray[i]))
	}
	if zerosum != 0 {
		t.Errorf("zerosum not zero (%d)", zerosum)
	}

	calculatedSizePerMap := calcMapBaseSize()

	if measuredMemPerMap < float64(calculatedSizePerMap)*(1.0-tolerance) {
		t.Errorf("measuredMemPerMap (%f) below tolerance of %f%% (%f of %d)", measuredMemPerMap, tolerance*100, float64(calculatedSizePerMap)*(1.0-tolerance), calculatedSizePerMap)
	}

	if measuredMemPerMap > float64(calculatedSizePerMap)*(1.0+tolerance) {
		t.Errorf("measuredMemPerMap (%f) above tolerance of %f%% (%f of %d)", measuredMemPerMap, tolerance*100, float64(calculatedSizePerMap)*(1.0+tolerance), calculatedSizePerMap)
	}
}

func generateRandomInt32Int32Map(size int) map[int32]int32 {
	randomMap := make(map[int32]int32, size)

	for i := 0; i < size; i++ {
		key := rand.Int31()
		value := rand.Int31()
		randomMap[key] = value
	}

	return randomMap
}

func TestGetTotalBucketCountUsingNOverflow(t *testing.T) {
	tests := []struct {
		mapInt32Int32       map[int32]int32
		expectedBucketCount uint64
	}{
		{
			mapInt32Int32:       generateRandomInt32Int32Map(5),
			expectedBucketCount: 1,
		},
		{
			mapInt32Int32:       generateRandomInt32Int32Map(9),
			expectedBucketCount: 2,
		},
		{
			mapInt32Int32:       generateRandomInt32Int32Map(50),
			expectedBucketCount: 8,
		},
		{
			mapInt32Int32:       generateRandomInt32Int32Map(415), // buckets x loadfactor -1 : 64*6.5 -1
			expectedBucketCount: 64,
		},
		{
			mapInt32Int32:       generateRandomInt32Int32Map(500), // loadfactor 6.5 -> requires ~77 buckets -> 128 buckets
			expectedBucketCount: 128,
		},
		{
			mapInt32Int32:       generateRandomInt32Int32Map(65536), // loadfactor 6.5 -> requires ~10,082 buckets -> 16,384 buckets
			expectedBucketCount: 16384,
		},
		{
			mapInt32Int32:       generateRandomInt32Int32Map(123456), // loadfactor 6.5 -> requires ~18,993 buckets -> 32,768 buckets
			expectedBucketCount: 32768,
		},
		{
			mapInt32Int32:       generateRandomInt32Int32Map(234567), // loadfactor 6.5 -> requires ~36,087 buckets -> 65,536 buckets
			expectedBucketCount: 65536,
		},
	}
	for _, test := range tests {

		totalBucketCount := getTotalBucketCountUsingNOverflow(reflect.Indirect(reflect.ValueOf(test.mapInt32Int32)))
		expectedCount := test.expectedBucketCount

		if totalBucketCount < expectedCount {
			t.Errorf("totalBucketCount (%d) below expectedCount (%d)", totalBucketCount, expectedCount)
		}

		if totalBucketCount > 2*expectedCount {
			// see func tooManyOverflowBuckets(noverflow uint16, B uint8) bool in golang-1.23/1.23.5-1/src/runtime/map.go
			// "too many" means (approximately) as many overflow buckets as regular buckets.
			t.Errorf("totalBucketCount (%d) above 2*expectedCount (%d)", totalBucketCount, expectedCount)
		}
	}
}

func TestCompareTotalBucketCountMethods(t *testing.T) {
	tests := []struct {
		mapEntryCount int
		iterations    int
		exactlyEqual  bool
	}{
		{
			mapEntryCount: 5,
			iterations:    100,
			exactlyEqual:  true,
		},
		{
			mapEntryCount: 9,
			iterations:    100,
			exactlyEqual:  true,
		},
		{
			mapEntryCount: 50,
			iterations:    100,
			exactlyEqual:  true,
		},
		{
			mapEntryCount: 500,
			iterations:    100,
			exactlyEqual:  true,
		},
		{
			mapEntryCount: 1234,
			iterations:    100,
			exactlyEqual:  true,
		},
		{
			mapEntryCount: 123456,
			iterations:    100,
			exactlyEqual:  true,
		},
		{
			mapEntryCount: 234567,
			iterations:    100,
			exactlyEqual:  false,
		},
	}
	for _, test := range tests {

		for nr := range test.iterations {
			m := generateRandomInt32Int32Map(test.mapEntryCount)

			totalBucketCountUsingNOverflow := getTotalBucketCountUsingNOverflow(reflect.Indirect(reflect.ValueOf(m)))
			totalBucketCountFollowingOverflowPointers := getTotalBucketCountFollowingOverflowPointers(reflect.Indirect(reflect.ValueOf(m)))

			if test.exactlyEqual {
				if totalBucketCountUsingNOverflow != totalBucketCountFollowingOverflowPointers {
					t.Errorf("UsingNOverflow != FollowingOverflowPointers (%d/%d) in iteration %d", totalBucketCountUsingNOverflow, totalBucketCountFollowingOverflowPointers, nr)
				}
			} else {
				// the map contains more than 65536 buckets. in this case noverflow contains only an estimate. there is no point in comparing them exaclty.
			}
		}
	}
}

func TestCalcBucketSize(t *testing.T) {
	type MyStruct struct {
		A int
		B int32
		C int64
		D string
	}

	tests := []struct {
		mapType           reflect.Type
		expectedSize32bit uint64
		expectedSize64bit uint64
	}{
		{
			mapType:           reflect.TypeOf(map[int]int{}),
			expectedSize32bit: 1*8 + 4*8 + 4*8 + 4,
			expectedSize64bit: 1*8 + 8*8 + 8*8 + 8,
		},
		{
			mapType:           reflect.TypeOf(map[int32]int32{}),
			expectedSize32bit: 1*8 + 4*8 + 4*8 + 4,
			expectedSize64bit: 1*8 + 4*8 + 4*8 + 8,
		},
		{
			mapType:           reflect.TypeOf(map[int64]int64{}),
			expectedSize32bit: 1*8 + 8*8 + 8*8 + 4,
			expectedSize64bit: 1*8 + 8*8 + 8*8 + 8,
		},
		{
			mapType:           reflect.TypeOf(map[int32]int64{}),
			expectedSize32bit: 1*8 + 4*8 + 8*8 + 4,
			expectedSize64bit: 1*8 + 4*8 + 8*8 + 8,
		},
		{
			mapType:           reflect.TypeOf(map[int64]int32{}),
			expectedSize32bit: 1*8 + 8*8 + 4*8 + 4,
			expectedSize64bit: 1*8 + 8*8 + 4*8 + 8,
		},
		{
			mapType:           reflect.TypeOf(map[int32]string{}),
			expectedSize32bit: 0, //?
			expectedSize64bit: 1*8 + 4*8 + 16*8 + 8,
		},
		{
			mapType:           reflect.TypeOf(map[string]int64{}),
			expectedSize32bit: 0, //?
			expectedSize64bit: 1*8 + 16*8 + 8*8 + 8,
		},
		{
			mapType:           reflect.TypeOf(map[string]string{}),
			expectedSize32bit: 0, //?
			expectedSize64bit: 1*8 + 16*8 + 16*8 + 8,
		},
		//        {reflect.TypeOf("a"), reflect.TypeOf("a"), 1*8 + uint64(unsafe.Sizeof(""))*internal_abi_MapBucketCount + uint64(unsafe.Sizeof(""))*internal_abi_MapBucketCount + 4, 1*internal_abi_MapBucketCount + uint64(unsafe.Sizeof(""))*internal_abi_MapBucketCount + uint64(unsafe.Sizeof(""))*internal_abi_MapBucketCount + 8},
		//        {reflect.TypeOf(MyStruct{}), reflect.TypeOf(MyStruct{}), 1*internal_abi_MapBucketCount + uint64(unsafe.Sizeof(MyStruct{}))*internal_abi_MapBucketCount + uint64(unsafe.Sizeof(MyStruct{}))*internal_abi_MapBucketCount + 4, 1*internal_abi_MapBucketCount + uint64(unsafe.Sizeof(MyStruct{}))*internal_abi_MapBucketCount + uint64(unsafe.Sizeof(MyStruct{}))*internal_abi_MapBucketCount + 8},
	}

	for _, test := range tests {
		expectedSize := test.expectedSize32bit
		if runtime.GOARCH != "386" && runtime.GOARCH != "arm" && runtime.GOARCH != "mips" && runtime.GOARCH != "mipsle" && runtime.GOARCH != "wasm" {
			expectedSize = test.expectedSize64bit
		}

		actualSize := calcBucketSize(test.mapType)
		if actualSize != expectedSize {
			t.Errorf("For map type %v -> %v, expected %d, but got %d", test.mapType.Key(), test.mapType.Elem(), expectedSize, actualSize)
		}
	}
}

func TestCompareCalcBucketSizeWithMeasurementInt32Int32(t *testing.T) {
	numberOfMaps := 1000000
	tolerance := 0.005
	mapsArray := make([]map[int32]int32, numberOfMaps)

	var startMem, endMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	for i := range numberOfMaps {
		mapsArray[i] = map[int32]int32{int32(1): int32(1)} // create a map with exactly 1 bucket
	}

	runtime.GC()
	runtime.ReadMemStats(&endMem)
	//usedMem := (endMem.HeapAlloc + endMem.StackInuse + endMem.StackSys) - (startMem.HeapAlloc + startMem.StackInuse + startMem.StackSys)
	usedMem := endMem.HeapAlloc - startMem.HeapAlloc
	measuredMemPerMap := float64(usedMem) / float64(numberOfMaps)

	// do something useless with the data to make sure it has not yet been garbage collected
	sum := int64(0)
	for i := range numberOfMaps {
		sum += int64(len(mapsArray[i]))
	}
	if sum != int64(numberOfMaps) {
		t.Errorf("sum (%d) wrong, should be %d", sum, numberOfMaps)
	}

	calculatedSizePerMap := calcMapBaseSize() + calcBucketSize(reflect.TypeOf(map[int32]int32{}))

	if measuredMemPerMap < float64(calculatedSizePerMap)*(1.0-tolerance) {
		t.Errorf("measuredMemPerMap (%f) below tolerance of %f%% (%f of %d)", measuredMemPerMap, tolerance*100, float64(calculatedSizePerMap)*(1.0-tolerance), calculatedSizePerMap)
	}

	if measuredMemPerMap > float64(calculatedSizePerMap)*(1.0+tolerance) {
		t.Errorf("measuredMemPerMap (%f) above tolerance of %f%% (%f of %d)", measuredMemPerMap, tolerance*100, float64(calculatedSizePerMap)*(1.0+tolerance), calculatedSizePerMap)
	}
}

func TestCompareCalcBucketSizeWithMeasurementStringInt64(t *testing.T) {
	numberOfMaps := 1000000
	tolerance := 0.005
	mapsArray := make([]map[string]int64, numberOfMaps)

	var startMem, endMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	for i := range numberOfMaps {
		mapsArray[i] = map[string]int64{"arghetjhaetjhajydfaehtytjhatahjtr": int64(1)} // create a map with exactly 1 bucket
	}

	runtime.GC()
	runtime.ReadMemStats(&endMem)
	//usedMem := (endMem.HeapAlloc + endMem.StackInuse + endMem.StackSys) - (startMem.HeapAlloc + startMem.StackInuse + startMem.StackSys)
	usedMem := endMem.HeapAlloc - startMem.HeapAlloc
	measuredMemPerMap := float64(usedMem) / float64(numberOfMaps)

	// do something useless with the data to make sure it has not yet been garbage collected
	sum := int64(0)
	for i := range numberOfMaps {
		sum += int64(len(mapsArray[i]))
	}
	if sum != int64(numberOfMaps) {
		t.Errorf("sum (%d) wrong, should be %d", sum, numberOfMaps)
	}

	calculatedSizePerMap := calcMapBaseSize() + calcBucketSize(reflect.TypeOf(map[string]int64{}))

	if measuredMemPerMap < float64(calculatedSizePerMap)*(1.0-tolerance) {
		t.Errorf("measuredMemPerMap (%f) below tolerance of %f%% (%f of %d)", measuredMemPerMap, tolerance*100, float64(calculatedSizePerMap)*(1.0-tolerance), calculatedSizePerMap)
	}

	if measuredMemPerMap > float64(calculatedSizePerMap)*(1.0+tolerance) {
		t.Errorf("measuredMemPerMap (%f) above tolerance of %f%% (%f of %d)", measuredMemPerMap, tolerance*100, float64(calculatedSizePerMap)*(1.0+tolerance), calculatedSizePerMap)
	}
}

func cloneMap[K comparable, V any](m map[K]V) map[K]V {
	result := make(map[K]V)
	for k, v := range m {
		result[k] = v
	}
	return result
}

func makeArray[K comparable, V any](_ map[K]V, length int) []map[K]V {
	return make([]map[K]V, length)
}

func TestSizeOfMapInt32Int32(t *testing.T) {
	numberOfMaps := 100000
	tolerance := 0.03

	original := generateRandomInt32Int32Map(50)

	mapsArray := makeArray(original, numberOfMaps)

	var startMem, endMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	for i := range numberOfMaps {
		mapsArray[i] = cloneMap(original)
	}

	runtime.GC()
	runtime.ReadMemStats(&endMem)
	//usedMem := (endMem.HeapAlloc + endMem.StackInuse + endMem.StackSys) - (startMem.HeapAlloc + startMem.StackInuse + startMem.StackSys)
	usedMem := endMem.HeapAlloc - startMem.HeapAlloc

	sum := uint64(0)
	for i := range numberOfMaps {
		m := mapsArray[i]
		s, err := Of(m)
		if err != nil {
			t.Errorf("Error getting size of map %d: %v", i, err)
		} else {
			sum += s
		}
	}

	if float64(usedMem) < float64(sum)*(1.0-tolerance) {
		t.Errorf("usedMem (%d) below tolerance of %f%% (%f of %d)", usedMem, tolerance*100, float64(sum)*(1.0-tolerance), sum)
	}

	if float64(usedMem) > float64(sum)*(1.0+tolerance) {
		t.Errorf("usedMem (%d) above tolerance of %f%% (%f of %d)", usedMem, tolerance*100, float64(sum)*(1.0+tolerance), sum)
	}
}

func TestSizeOfMap(t *testing.T) {
	tests := []struct {
		name     string
		original map[int32]int32
	}{
		{name: "empty_map[int32]int32", original: map[int32]int32{}},
		{name: "1_map[int32]int32", original: generateRandomInt32Int32Map(1)},
		{name: "4_map[int32]int32", original: generateRandomInt32Int32Map(4)},
		{name: "9_map[int32]int32", original: generateRandomInt32Int32Map(9)},
		{name: "10_map[int32]int32", original: generateRandomInt32Int32Map(10)},
		{name: "11_map[int32]int32", original: generateRandomInt32Int32Map(11)},
		{name: "50_map[int32]int32", original: generateRandomInt32Int32Map(50)},
	}

	numberOfMaps := 100000
	tolerance := 0.01

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapsArray := makeArray(test.original, numberOfMaps)

			var startMem, endMem runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startMem)

			for i := range numberOfMaps {
				mapsArray[i] = cloneMap(test.original)
			}

			runtime.GC()
			runtime.ReadMemStats(&endMem)
			//usedMem := (endMem.HeapAlloc + endMem.StackInuse + endMem.StackSys) - (startMem.HeapAlloc + startMem.StackInuse + startMem.StackSys)
			usedMem := endMem.HeapAlloc - startMem.HeapAlloc

			sum := uint64(0)
			for i := range numberOfMaps {
				m := mapsArray[i]
				s, err := Of(m)
				if err != nil {
					t.Errorf("Error getting size of map %d: %v", i, err)
				} else {
					sum += s
				}
			}

			if float64(usedMem) < float64(sum)*(1.0-tolerance) {
				t.Errorf("usedMem (%d) below tolerance of %f%% (%f of %d)", usedMem, tolerance*100, float64(sum)*(1.0-tolerance), sum)
			}

			if float64(usedMem) > float64(sum)*(1.0+tolerance) {
				t.Errorf("usedMem (%d) above tolerance of %f%% (%f of %d)", usedMem, tolerance*100, float64(sum)*(1.0+tolerance), sum)
			}
		})
	}
}

/*
func TestSizeOfMapOld(t *testing.T) {
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
*/
