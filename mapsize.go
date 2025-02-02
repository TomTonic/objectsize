package objectsize

import (
	"reflect"
	"unsafe"
)

func sizeOfMapTypeParam(t reflect.Type) uint64 {
	return uint64(t.Size())
}

func sizeOfMapBucket(v reflect.Value) uint64 {
	var tophasharraybasetype uint8
	var ptr uintptr
	result := uint64(unsafe.Sizeof(tophasharraybasetype))
	mapType := v.Type()
	keyType := mapType.Key()
	result += sizeOfMapTypeParam(keyType)
	valueType := mapType.Elem()
	result += sizeOfMapTypeParam(valueType)
	result *= internal_abi_MapBucketCount
	result += uint64(unsafe.Sizeof(ptr))
	return result
}

func sizeOfMap(v reflect.Value, cache map[uintptr]bool) (uint64, error) {
	var u uintptr
	sum := uint64(unsafe.Sizeof(u))

	// Use reflection to get the pointer to the map's internal structure
	mapPointer := unsafe.Pointer(v.Pointer())

	// Cast the map pointer to the hmap struct
	hmapStruct := (*hmap)(mapPointer)

	// sum, err := sizeOfStruct(reflect.Indirect(reflect.ValueOf(hmapStruct)), cache)
	sum += uint64(unsafe.Sizeof(hmapStruct.count))
	sum += uint64(unsafe.Sizeof(hmapStruct.flags))
	sum += uint64(unsafe.Sizeof(hmapStruct.B))
	sum += uint64(unsafe.Sizeof(hmapStruct.noverflow))
	sum += uint64(unsafe.Sizeof(hmapStruct.hash0))
	sum += uint64(unsafe.Sizeof(hmapStruct.nevacuate))

	sum += uint64(unsafe.Sizeof(hmapStruct.buckets))
	if hmapStruct.buckets != nil {
		bucketarraysize := uint64(1) << hmapStruct.B
		sum += uint64(unsafe.Sizeof(u)) * bucketarraysize
	}

	sum += uint64(unsafe.Sizeof(hmapStruct.oldbuckets))
	if hmapStruct.oldbuckets != nil {
		oldbucketarraysize := uint64(1) << (hmapStruct.B - 1)
		sum += uint64(unsafe.Sizeof(u)) * oldbucketarraysize
	}

	if hmapStruct.extra == nil {
		sum += uint64(unsafe.Sizeof(hmapStruct.extra))
	} else {
		s, err := sizeOfStruct(reflect.Indirect(reflect.ValueOf(*hmapStruct.extra)), cache)
		if err != nil {
			return sum, err
		}
		sum += s
	}

	// Iterate through the map using reflection
	for _, key := range v.MapKeys() {
		s1, err1 := sizeOf(reflect.Indirect(key), cache)
		if err1 != nil {
			return sum, err1
		}
		sum += s1
		value := v.MapIndex(key)
		s2, err2 := sizeOf(reflect.Indirect(value), cache)
		if err2 != nil {
			return sum, err2
		}
		sum += s2
	}

	/*
		const MapBucketCount = 8
		hmap := reflect.ValueOf(m.val).Elem()
		B := hmap.FieldByName("B").Uint()
		buckets := hmap.FieldByName("buckets").Pointer()
		oldbuckets := hmap.FieldByName("oldbuckets").Pointer()
		flags := hmap.FieldByName("flags").Uint()
		inttype := hmap.FieldByName("hash0").Type()
		cnt := 0

		for bucket := uintptr(0); bucket < 1<<B; bucket++ {
			bp := unsafe.Pointer(buckets + bucket)
			if oldbuckets != 0 {
				oldbucket := bucket & (1<<(B-1) - 1)
				oldbp := unsafe.Pointer(oldbuckets + oldbucket)
				oldb := reflect.NewAt(inttype, oldbp).Elem()
				if oldb.FieldByName("overflow").Uint()&1 == 0 {
					if bucket >= 1<<(B-1) {
						continue
					}
					bp = oldbp
				}
			}
			for bp != nil {
				b := reflect.NewAt(inttype, bp).Elem()
				for i := 0; i < MapBucketCount; i++ {
					if b.FieldByName("tophash").Index(i).Uint() != 0 {
						k := b.FieldByName("keys").Index(i)
						v := b.FieldByName("values").Index(i)
						if flags&1 != 0 {
							k = k.Elem()
						}
						if flags&2 != 0 {
							v = v.Elem()
						}
						fmt.Printf("%d: %v\n", cnt, k)
						fmt.Printf("%d: %v\n", cnt+1, v)
						cnt += 2
					}
				}
				bp = unsafe.Pointer(b.FieldByName("overflow").Pointer())
			}
		}
	*/

	return sum, nil
}
