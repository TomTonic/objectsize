package objectsize

import (
	"testing"
)

type nodeType struct {
	value int       // 4 - int
	next  *nodeType // 8 - pointer
	// 4 - padding
}

type typeWithFunc struct {
	a   int
	foo func(param int)
	b   int
}

func TestOf(t *testing.T) {
	s := make([]string, 1)           // 24
	ss := make([][]string, 100, 100) // 100 * 24 + 24
	s[0] = "1234"                    // 16 + 4
	for i := range ss {
		ss[i] = s
	} // 24 + 16 + 4 + 99 * 24 + 24 = 2444
	n2 := nodeType{
		value: 2,
	}
	n1 := nodeType{
		value: 1,
		next:  &n2,
	}
	n0 := nodeType{
		value: 0,
		next:  &n1,
	}
	n2.next = &n1
	n3 := nodeType{
		value: 3,
	}
	n4 := nodeType{
		value: 4,
		next:  &n3,
	}

	tests := []struct {
		name string
		v    interface{}
		want uint64
	}{
		{
			name: "Array",
			v:    [3]int32{1, 2, 3}, // 3 * 4  = 12
			want: 12,
		},
		{
			name: "Slice",
			v:    make([]int64, 2, 5), // 5 * 8 + 24 = 64
			want: 64,
		},
		{
			name: "String",
			v:    "ABCdef", // 6 + 16 = 22
			want: 22,
		},
		{
			name: "Two Strings",
			v:    [2]string{"ABC", "def"}, // 2 * (3 + 16) = 38
			want: 38,
		},
		{
			name: "Two Equal Strings",
			v:    [2]string{"ABC", "ABC"}, // 2 * (3 + 16) -3 = 38 -3 = 35
			want: 35,
		},
		/*		{
					name: "Map",
					// (8 + 3 + 16) + (8 + 4 + 16) = 55
					// 55 + 8 + 10.79 * 2 = 84
					v:    map[int64]string{0: "ABC", 1: "DEFG"},
					want: 84,
				},
		*/{
			name: "Struct",
			v: struct {
				slice     []int64
				array     [2]bool
				structure struct {
					i int8
					s string
				}
			}{
				slice: []int64{12345, 67890}, // 2 * 8 + 24 = 40
				array: [2]bool{true, false},  // 2 * 1 = 2
				structure: struct {
					i int8
					s string
				}{
					i: 5,     // 1
					s: "abc", // 3 * 1 + 16 = 19
				}, // 20 + 7 (padding) = 27
			}, // 40 + 2 + 27 = 69 + 6 (padding) = 75
			want: 75,
		},
		{
			name: "Struct With Func",
			v: typeWithFunc{
				a:   5,   // 8 (4+padding)
				foo: nil, // 8
				b:   13,  // 8 (4+padding)
			},
			want: 24,
		},
		{
			name: "Slice of strings slices (slice of cloned slices)",
			v:    ss,
			want: 2444,
		},
		{
			name: "Struct with the same slice value in two fields",
			v: struct {
				s1 []string // 24
				s2 []string // 24
			}{
				s1: s, // + 16 + 4
				s2: s, // + 0 (the same)
			}, // 2 * 24 + 16 + 4 = 68
			want: 68,
		},
		{
			name: "Single node",
			v:    n3,
			want: 16,
		},
		{
			name: "Two nodes",
			v:    n4,
			want: 32,
		},
		{
			name: "Three nodes with cyclic structure",
			v:    n0,
			want: 48,
		},
		{
			name: "Interface in Array",
			v:    [4]interface{}{1, 2, 3, 4}, // 4 * (8+16) = 96
			want: 96,
		},
		{
			name: "Interface in Slice",
			v:    []interface{}{12345, 67890}, // 2 * (8+16) + 24 = 72
			want: 72,
		},
		{
			name: "Interface in Struct",
			v: struct {
				a uint64
				b interface{}
				c interface{}
			}{
				a: 5,     // 8
				b: 13,    // 16 + 8 = 24
				c: "abc", // 16 + 3 * 1 + 16 = 35
			},
			want: 67,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := Of(tt.v); err != nil || got != tt.want {
				if err != nil {
					t.Errorf("Of() returned an error: %v", err)
				} else {
					t.Errorf("Of() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestOfInvalid(t *testing.T) {
	var interfaceVar interface{}
	tests := []struct {
		name string
		v    interface{}
	}{
		{
			name: "Single",
			v:    interfaceVar,
		},
		{
			name: "Array",
			v:    [3]interface{}{1, 2, interfaceVar},
		},
		{
			name: "Slice",
			v:    []interface{}{1, 2, interfaceVar},
		},
		{
			name: "Struct",
			v: struct {
				a int
				b interface{}
			}{
				a: 5,
				b: interfaceVar,
			},
		},
		{
			name: "Pointer",
			v: struct {
				a *interface{}
			}{
				a: &interfaceVar,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Of(tt.v); err == nil {
				t.Errorf("Of() returned no error")
			}
		})
	}
}
