package commands

import (
	"fmt"
)

func ExampleMerge() {
	fmt.Println(merge(map[string]string{"A": "1"}, []string{"B=2"}))
	// Output: [B=2 A=1]
}
