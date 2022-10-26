package clock

import "time"

// Default is the default clock.
var Default = Func(time.Now)

// Clock tells you the current time.
type Clock interface {
	Now() time.Time
}

// Func is a function that returns a time.
type Func func() time.Time

// Now() ensures that Func satisfies the Clock interface.
func (fn Func) Now() time.Time { return fn() }
