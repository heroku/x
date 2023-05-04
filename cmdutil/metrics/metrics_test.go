package metrics

import (
	"testing"

	"github.com/heroku/x/go-kit/metrics/testmetrics"
)

func TestReportPanic(t *testing.T) {
	mp := testmetrics.NewProvider(t)

	defer func() {
		if p := recover(); p == nil {
			t.Fatal("expected ReportPanic to repanic")
		}
		mp.CheckObservationCount("panic", 1)
	}()

	func() {
		defer ReportPanic(mp)

		panic("test message")
	}()
}
