# size - calculates variable's memory consumption at runtime

Sometimes you may need a tool to calculate the size of an object (-tree) in your Go program at runtime. This package makes an attempt to do so.
The original package is based on `binary.Size()` from Go standard library.

Features:

- supports non-fixed size variables and struct fields: `struct`, `int`, `slice`, `string`, ~~map~~;
- supports complex types including structs with non-fixed size fields;
- supports all basic types (numbers, bool);
- supports `chan` and `interface`;
- supports pointers;
- implements infinite recursion detection (i.e. pointer inside struct field references to parent struct).

## Usage example

```Golang
package main

import (
    "fmt"

    "github.com/TomTonic/objectsize"
)

func main() {
    a := struct {
        a int
        b string
        c bool
        d int32
        e []byte
        f [3]int64
    }{
        a: 10,                    // 8 bytes
        b: "Text",                // 16 (string itself) + 4 = 20 bytes
        c: true,                  // 1 byte
        d: 25,                    // 4 bytes
        e: []byte{'c', 'd', 'e'}, // 24 (slice itself) + 3 = 27 bytes
        f: [3]int64{1, 2, 3},     // 3 * 8 = 24 bytes
    } // 84 + 3 (padding) = 87 bytes

    fmt.Println(size.Of(a))
}

// Output: 87
```
