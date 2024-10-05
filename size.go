// Package objectsize implements run-time calculation of size of an object (-tree) in Go.
// The source code is based on the "binary.Size()" function from Go standard library.
// size.Of() omits size of slices, arrays and maps containers itself (24, 24 and 8 bytes).
// When counting maps separate calculations are done for keys and values.
package objectsize

import (
	"errors"
	"reflect"
	"unsafe"
)

type stringHeader struct {
	data uintptr
	len  int
}

// Of returns the size of 'v' in bytes.
// Returns 0 and error!=nil if there is an error during calculation.
func Of(v interface{}) (uint64, error) {
	// Cache with every visited pointer so we don't count two pointers
	// to the same memory twice.
	cache := make(map[uintptr]bool)
	result, err := sizeOf(reflect.Indirect(reflect.ValueOf(v)), cache)
	return result, err
}

// sizeOf returns the number of bytes the actual data represented by v occupies in memory.
// If there is an error, sizeOf returns -1.
func sizeOf(v reflect.Value, cache map[uintptr]bool) (uint64, error) {
	switch v.Kind() {

	case reflect.Array:
		return sizeOfArray(v, cache)

	case reflect.Slice:
		return sizeOfSlice(v, cache)

	case reflect.Struct:
		return sizeOfStruct(v, cache)

	case reflect.String:
		return sizeOfString(v, cache)

	case reflect.Pointer:
		return sizeOfPointer(v, cache)

	case reflect.Bool,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Int, reflect.Uint,
		reflect.Chan,
		reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.Func:
		return uint64(v.Type().Size()), nil
		/*
			// the map size calculation is invalid as "10.79" only holds true for maps that are
			// a) completely full AND
			// b) keys have 8 byte size AND
			// c) items have 8 bytes size AND
			// d) no pointers are involved
			case reflect.Map:
				// return 0 if this node has been visited already (infinite recursion)
				if cache[v.Pointer()] {
					return 0, nil
				}
				cache[v.Pointer()] = true
				var sum uint64
				keys := v.MapKeys()
				for i := range keys {
					val := v.MapIndex(keys[i])
					// calculate size of key and value separately
					sv, err := sizeOf(val, cache)
					if err != nil {
						return 0, err
					}
					sum += sv
					sk, err := sizeOf(keys[i], cache)
					if err != nil {
						return 0, err
					}
					sum += sk
				}
				// Include overhead due to unused map buckets.  10.79 comes
				// from https://golang.org/src/runtime/map.go.
				return (sum + uint64(v.Type().Size()) + uint64(float64(len(keys))*10.79)), nil
		*/

	case reflect.Interface:
		return sizeOfInterface(v, cache)
	}

	// can currently only be reflect.Map or reflect.Invalid or reflect.UnsafePointer, see type.go
	return 0, errors.New("unimplemented kind: " + v.Kind().String())
}

func sizeOfInterface(v reflect.Value, cache map[uintptr]bool) (uint64, error) {
	interfaceSize := uint64(v.Type().Size())
	s, err := sizeOf(v.Elem(), cache)
	if err != nil {
		return 0, err
	}
	return s + interfaceSize, nil
}

func sizeOfPointer(v reflect.Value, cache map[uintptr]bool) (uint64, error) {
	pointerSize := uint64(v.Type().Size())
	if v.IsNil() {
		return pointerSize, nil
	}
	if cache[v.Pointer()] {
		// we already visited this object, do not visit it again
		return pointerSize, nil
	}
	cache[v.Pointer()] = true
	s, err := sizeOf(reflect.Indirect(v), cache)
	if err != nil {
		return 0, err
	}
	return s + pointerSize, nil
}

func sizeOfString(v reflect.Value, cache map[uintptr]bool) (uint64, error) {
	stringSize := uint64(v.Type().Size())
	s := v.String()
	data := (*stringHeader)(unsafe.Pointer(&s)).data
	if cache[data] {
		// there is a backing data array that has already been used. don't count it again
		return stringSize, nil
	}
	cache[data] = true
	return uint64(len(s)) + stringSize, nil
}

func sizeOfStruct(v reflect.Value, cache map[uintptr]bool) (uint64, error) {
	var sum uint64
	for i, n := 0, v.NumField(); i < n; i++ {
		s, err := sizeOf(v.Field(i), cache)
		if err != nil {
			return 0, err
		}
		sum += s
	}

	padding := uint64(v.Type().Size())
	for i, n := 0, v.NumField(); i < n; i++ {
		padding -= uint64(v.Field(i).Type().Size())
	}

	return (sum + padding), nil
}

func sizeOfSlice(v reflect.Value, cache map[uintptr]bool) (uint64, error) {
	sliceSize := uint64(v.Type().Size())
	if cache[v.Pointer()] {
		return sliceSize, nil
	}
	cache[v.Pointer()] = true

	var sum uint64
	for i := 0; i < v.Len(); i++ {
		s, err := sizeOf(v.Index(i), cache)
		if err != nil {
			return 0, err
		}
		sum += s
	}

	sum += uint64(v.Cap()-v.Len()) * uint64(v.Type().Elem().Size())
	result := sum + sliceSize
	return result, nil
}

func sizeOfArray(v reflect.Value, cache map[uintptr]bool) (uint64, error) {
	var sum uint64
	for i := 0; i < v.Len(); i++ {
		s, err := sizeOf(v.Index(i), cache)
		if err != nil {
			return 0, err
		}
		sum += s
	}
	sum += uint64(v.Cap()-v.Len()) * uint64(v.Type().Elem().Size())
	return sum, nil
}
