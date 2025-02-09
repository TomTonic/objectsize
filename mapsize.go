package objectsize

import (
	"reflect"
	"unsafe"
)

func calcMapBaseSize() uint64 {
	var i int
	var up unsafe.Pointer
	var ptr uintptr
	var pme *mapextra
	result := uint64(unsafe.Sizeof(i))   // hmap.count
	result += 1                          // hmap.flags
	result += 1                          // hmap.B
	result += 2                          // hmap.noverflow
	result += 4                          // hmap.hash0
	result += uint64(unsafe.Sizeof(up))  // hmap.buckets
	result += uint64(unsafe.Sizeof(up))  // hmap.oldbuckets
	result += uint64(unsafe.Sizeof(ptr)) // hmap.nevacuate
	result += uint64(unsafe.Sizeof(pme)) // hmap.extra
	return result
}

func calcBucketSize(mapType reflect.Type) uint64 {
	keyType := mapType.Key()
	valueType := mapType.Elem()

	result := uint64(1) * internal_abi_MapBucketCount                // bmap.tophash[]
	result += uint64(keyType.Size()) * internal_abi_MapBucketCount   // bmap."keys[]"
	result += uint64(valueType.Size()) * internal_abi_MapBucketCount // bmap."values[]"
	result += uint64(unsafe.Sizeof(uintptr(0)))                      // bmap."overflow"

	return result
}

func calcOverflowPtrOffset(mapType reflect.Type) uintptr {
	keyType := mapType.Key()
	valueType := mapType.Elem()

	result := uintptr(1) * internal_abi_MapBucketCount                // bmap.tophash[]
	result += uintptr(keyType.Size()) * internal_abi_MapBucketCount   // bmap."keys[]"
	result += uintptr(valueType.Size()) * internal_abi_MapBucketCount // bmap."values[]"
	// next field is bmap."overflow"
	return result
}

func getTotalBucketCountUsingNOverflow(v reflect.Value) uint64 {
	// see func (h *hmap) incrnoverflow() from golang-1.23/1.23.5-1/src/runtime/map.go

	// Use reflection to get the pointer to the map's internal structure
	mapPointer := unsafe.Pointer(v.Pointer())

	// Cast the map pointer to the hmap struct
	hmapPtr := (*hmap)(mapPointer)

	result := uint64(0)
	if hmapPtr.buckets != nil {
		result += uint64(1) << hmapPtr.B
	}
	if hmapPtr.oldbuckets != nil {
		result += uint64(1) << (hmapPtr.B - 1)
	}
	if hmapPtr.B < 16 {
		result += uint64(hmapPtr.noverflow)
	} else {
		multiplier := uint64(1) << (hmapPtr.B - 15)
		result += uint64(hmapPtr.noverflow) * multiplier
	}
	return result
}

func nextOverflowBucket(bmapPtr *bmap, overflowPtrOffset uintptr) *bmap {
	// see func (b *bmap) overflow(t *maptype) *bmap from golang-1.23/1.23.5-1/src/runtime/map.go
	return *(**bmap)(unsafe.Add(unsafe.Pointer(bmapPtr), overflowPtrOffset))
}

func getOverflowChainLength(bmapPtr *bmap, overflowPtrOffset uintptr, limit uint64) uint64 {
	result := uint64(0)
	current := bmapPtr
	// limit is our safetybelt: if something goes wrong with the data structure we won't jump around in memory forever...
	for current != nil && result < limit {
		result++
		current = nextOverflowBucket(current, overflowPtrOffset)
	}
	return result
}

func getTotalBucketCountFollowingOverflowPointers(v reflect.Value) uint64 {
	var ptr uintptr

	// Use reflection to get the pointer to the map's internal structure
	mapPointer := unsafe.Pointer(v.Pointer())

	// Cast the map pointer to the hmap struct
	hmapPtr := (*hmap)(mapPointer)

	bucketSize := calcBucketSize(v.Type())
	overflowPtrOffset := uintptr(bucketSize) - uintptr(unsafe.Sizeof(ptr))

	result := uint64(0)
	if hmapPtr.buckets != nil {
		result += countBuckets(hmapPtr.buckets, hmapPtr.B, bucketSize, overflowPtrOffset)
	}
	if hmapPtr.oldbuckets != nil {
		result += countBuckets(hmapPtr.oldbuckets, hmapPtr.B-1, bucketSize, overflowPtrOffset)
	}

	return result
}

func countBuckets(p unsafe.Pointer, B uint8, bucketSize uint64, overflowPtrOffset uintptr) uint64 {
	result := uint64(0)
	count := uint64(1) << B
	for bucketIndex := range uintptr(count) {
		bucketPtr := (*bmap)(unsafe.Add(p, bucketIndex*uintptr(bucketSize)))
		length := getOverflowChainLength(bucketPtr, overflowPtrOffset, count)
		result += length
	}
	return result
}

func sizeOfExtra(extra *mapextra) uint64 {
	if extra == nil {
		return 0
	}
	//sum := uint64(0)
	//sum += uint64(unsafe.Sizeof(extra.overflow))
	//sum += uint64(unsafe.Sizeof(extra.oldoverflow))
	//sum += uint64(unsafe.Sizeof(extra.nextOverflow))
	sum := 3 * uint64(unsafe.Sizeof(uintptr(0)))
	if extra.overflow != nil {
		sum += uint64(len(*extra.overflow)) * uint64(unsafe.Sizeof(uintptr(0)))
	}
	if extra.oldoverflow != nil {
		sum += uint64(len(*extra.oldoverflow)) * uint64(unsafe.Sizeof(uintptr(0)))
	}
	return sum
}

func sizeOfMap(v reflect.Value, cache map[uintptr]bool) (uint64, error) {
	result := uint64(0)

	result += calcMapBaseSize()

	bucketSize := calcBucketSize(v.Type())
	totalBucketCount := getTotalBucketCountFollowingOverflowPointers(v)

	result += bucketSize * totalBucketCount

	// Use reflection to get the pointer to the map's internal structure
	mapPointer := unsafe.Pointer(v.Pointer())
	// Cast the map pointer to the hmap struct
	hmapPtr := (*hmap)(mapPointer)

	result += sizeOfExtra(hmapPtr.extra)

	if hmapPtr.extra != nil && hmapPtr.extra.nextOverflow != nil {
		// there are more reserve buckets from the initial array allocation
		// than we found by counting the buckets following the Overflow pointers
		// -> walk them until we find one that has an non-nil nextOverflow ptr
		// (see func (h *hmap) newoverflow(t *maptype, b *bmap) *bmap of golang-1.23/1.23.5-1/src/runtime/map.go)
		bucketSize := calcBucketSize(v.Type())
		overflowPtrOffset := calcOverflowPtrOffset(v.Type())
		var ovf *bmap
		ovf = hmapPtr.extra.nextOverflow
		nextptrval := nextOverflowBucket(ovf, overflowPtrOffset)
		reserveBucketsCounter := uint64(1)
		for nextptrval == nil {
			reserveBucketsCounter++
			// We're not at the end of the preallocated overflow buckets. Bump the pointer.
			ovf = (*bmap)(unsafe.Add(unsafe.Pointer(ovf), uintptr(bucketSize)))
			nextptrval = nextOverflowBucket(ovf, overflowPtrOffset)
		}
		result += reserveBucketsCounter * bucketSize
	}

	// TODO: Descend into object trees if keys/values are not stored directly in the hashmap
	/*
		// Iterate through the map using reflection
		for _, key := range v.MapKeys() {
			s1, err1 := sizeOf(reflect.Indirect(key), cache)
			if err1 != nil {
				return result, err1
			}
			result += s1
			value := v.MapIndex(key)
			s2, err2 := sizeOf(reflect.Indirect(value), cache)
			if err2 != nil {
				return result, err2
			}
			result += s2
		}
	*/

	return result, nil
}
