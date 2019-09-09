package tickgroup

import (
	"context"
	"fmt"
	"time"
)

// This example uses the tickgroup package to create a simple 5 second timer.
func Example() {
	ctx, cancel := context.WithCancel(context.Background())

	// A tickgroup tg with a cancel context is initialized.
	tg := New(ctx)

	var i int
	tg.Go(time.Second, func() error {
		if i > 4 {
			cancel()
			return nil
		}
		// i is incremeneted every 1 second.
		i++
		fmt.Println(i)
		return nil
	})

	// After 5 seconds the context is canceled and Wait() returns a nil value, which ceases the process.
	if err := tg.Wait(); err != nil {
		fmt.Println(err)
	}

	// Output:
	// 1
	// 2
	// 3
	// 4
	// 5
}
