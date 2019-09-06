# tickgroup

## Overview

A tickgroup is a collection of goroutines that are spawned to call `f()`
every set interval.  A  tickgroup will stop spawning subtasks, when `ctx` is
done.  If subtask `f()` encounters an error, the tickgroup is canceled and
the error is returned.

## Example

This example uses the tickgroup package to create a simple 5 second timer.  A
tickgroup `tg` with a cancel context is initialized.  Inside of `Go()`, `i`
is incremented every 1 second; after 5 seconds the context is canceled and
`Wait()` returns a nil value, which ceases the proccess.

```go
ctx, cancel := context.WithCancel(context.Background())
tg := tickgroup.New(ctx)

var i int
tg.Go(time.Second, func() error {
	if i > 4 {
		cancel()
		return nil
	}
	i++
	return nil
})

if err := tg.Wait(); err != nil {
	log.Print(err)
}
```
