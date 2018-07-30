package fenwick_test

import (
	"fmt"
	"github.com/yourbasic/fenwick"
)

// Compute the sum of the first 4 elements in a list.
func Example() {
	a := fenwick.New(1, 2, 3, 4, 5)
	fmt.Println(a.Sum(4))
	// Output: 10
}
