package lowcard

import (
	"fmt"
	"testing"

	xmetrics "github.com/heroku/x/go-kit/metrics"
	"github.com/heroku/x/go-kit/metrics/testmetrics"
)

var (
	testBaseName      = "i.am.a"
	testCounterName   = fmt.Sprintf("%s.counter", testBaseName)
	testGaugeName     = fmt.Sprintf("%s.gauge", testBaseName)
	testHistogramName = fmt.Sprintf("%s.histogram", testBaseName)
	testLabelName     = "label"
)

func TestWrappedProvider(t *testing.T) {
	t.Run("counter label skipped", func(t *testing.T) {
		mp := NewLowCardinalityWrappedProvider(testmetrics.NewProvider(t), []string{})
		defer mp.Stop()

		c := mp.NewCounter(testCounterName)
		c.Add(1)

		var labels []string

		checkCounter(t, mp, testCounterName, 1, labels...)

		c = c.With(testBaseName, testLabelName)
		c.Add(1)

		checkCounter(t, mp, testCounterName, 2, labels...)
	})

	t.Run("gauge label skipped", func(t *testing.T) {
		mp := NewLowCardinalityWrappedProvider(testmetrics.NewProvider(t), []string{})
		defer mp.Stop()

		g := mp.NewGauge(testGaugeName)
		g.Add(1)

		var labels []string

		checkGauge(t, mp, testGaugeName, 1, labels...)

		g = g.With(testBaseName, testLabelName)
		g.Add(1)

		checkGauge(t, mp, testGaugeName, 2, labels...)
	})

	t.Run("histogram label skipped", func(t *testing.T) {
		mp := NewLowCardinalityWrappedProvider(testmetrics.NewProvider(t), []string{})
		defer mp.Stop()

		g := mp.NewHistogram(testHistogramName, 50)
		g.Observe(float64(1))

		var labels []string
		var obs = []float64{1}

		checkHistogram(t, mp, testHistogramName, obs, labels...)

		g = g.With(testBaseName, testLabelName)

		g.Observe(float64(1))
		obs = append(obs, float64(1))

		checkHistogram(t, mp, testHistogramName, obs, labels...)
	})

	t.Run("explicit histogram label skipped", func(t *testing.T) {
		mp := NewLowCardinalityWrappedProvider(testmetrics.NewProvider(t), []string{})
		defer mp.Stop()

		g := mp.NewExplicitHistogram(testHistogramName, xmetrics.TenSecondDistribution)
		g.Observe(float64(1))

		var labels []string
		var obs = []float64{1}

		checkHistogram(t, mp, testHistogramName, obs, labels...)

		g = g.With(testBaseName, testLabelName)

		g.Observe(float64(1))
		obs = append(obs, float64(1))

		checkHistogram(t, mp, testHistogramName, obs, labels...)
	})

	t.Run("counter label added", func(t *testing.T) {
		mp := NewLowCardinalityWrappedProvider(testmetrics.NewProvider(t), []string{testBaseName})
		defer mp.Stop()

		c := mp.NewCounter(testCounterName)
		c.Add(1)

		var labels []string

		checkCounter(t, mp, testCounterName, 1, labels...)

		labels = append(labels, testBaseName, testLabelName)
		c = c.With(labels...)

		c.Add(1)

		checkCounter(t, mp, testCounterName, 1, labels...)
	})

	t.Run("gauge label added", func(t *testing.T) {
		mp := NewLowCardinalityWrappedProvider(testmetrics.NewProvider(t), []string{testBaseName})
		defer mp.Stop()

		g := mp.NewGauge(testGaugeName)
		g.Add(1)

		var labels []string

		checkGauge(t, mp, testGaugeName, 1, labels...)

		labels = append(labels, testBaseName, testLabelName)
		g = g.With(labels...)

		g.Add(1)

		checkGauge(t, mp, testGaugeName, 1, labels...)
	})

	t.Run("histogram label added", func(t *testing.T) {
		mp := NewLowCardinalityWrappedProvider(testmetrics.NewProvider(t), []string{testBaseName})
		defer mp.Stop()

		h := mp.NewHistogram(testHistogramName, 50)
		h.Observe(float64(1))

		var labels []string
		var obs = []float64{1}

		checkHistogram(t, mp, testHistogramName, obs, labels...)

		labels = append(labels, testBaseName, testLabelName)
		h = h.With(labels...)

		h.Observe(float64(1))

		checkHistogram(t, mp, testHistogramName, obs, labels...)
	})

	t.Run("explicit histogram label added", func(t *testing.T) {
		mp := NewLowCardinalityWrappedProvider(testmetrics.NewProvider(t), []string{testBaseName})
		defer mp.Stop()

		h := mp.NewExplicitHistogram(testHistogramName, xmetrics.TenSecondDistribution)
		h.Observe(float64(1))

		var labels []string
		var obs = []float64{1}

		checkHistogram(t, mp, testHistogramName, obs, labels...)

		labels = append(labels, testBaseName, testLabelName)
		h = h.With(labels...)

		h.Observe(float64(1))

		checkHistogram(t, mp, testHistogramName, obs, labels...)
	})

	t.Run("multiple counter labels added", func(t *testing.T) {
		mp := NewLowCardinalityWrappedProvider(testmetrics.NewProvider(t), []string{testBaseName})
		defer mp.Stop()

		c := mp.NewCounter(testCounterName)
		c.Add(1)

		var labels []string

		checkCounter(t, mp, testCounterName, 1, labels...)

		labels = append(labels, testBaseName, testLabelName, testBaseName, testLabelName)
		c = c.With(labels...)

		c.Add(1)

		checkCounter(t, mp, testCounterName, 1, labels...)
	})

	t.Run("multiple gauge labels added", func(t *testing.T) {
		mp := NewLowCardinalityWrappedProvider(testmetrics.NewProvider(t), []string{testBaseName})
		defer mp.Stop()

		g := mp.NewGauge(testGaugeName)
		g.Add(1)

		var labels []string

		checkGauge(t, mp, testGaugeName, 1, labels...)

		labels = append(labels, testBaseName, testLabelName, testBaseName, testLabelName)
		g = g.With(labels...)

		g.Add(1)

		checkGauge(t, mp, testGaugeName, 1, labels...)
	})

	t.Run("multiple histogram labels added", func(t *testing.T) {
		mp := NewLowCardinalityWrappedProvider(testmetrics.NewProvider(t), []string{testBaseName})
		defer mp.Stop()

		h := mp.NewHistogram(testHistogramName, 50)
		h.Observe(float64(1))

		var labels []string
		var obs = []float64{1}

		checkHistogram(t, mp, testHistogramName, obs, labels...)

		labels = append(labels, testBaseName, testLabelName, testBaseName, testLabelName)
		h = h.With(labels...)

		h.Observe(float64(1))

		checkHistogram(t, mp, testHistogramName, obs, labels...)
	})

	t.Run("multiple explicit histogram labels added", func(t *testing.T) {
		mp := NewLowCardinalityWrappedProvider(testmetrics.NewProvider(t), []string{testBaseName})
		defer mp.Stop()

		h := mp.NewExplicitHistogram(testHistogramName, xmetrics.TenSecondDistribution)
		h.Observe(float64(1))

		var labels []string
		var obs = []float64{1}

		checkHistogram(t, mp, testHistogramName, obs, labels...)

		labels = append(labels, testBaseName, testLabelName, testBaseName, testLabelName)
		h = h.With(labels...)

		h.Observe(float64(1))

		checkHistogram(t, mp, testHistogramName, obs, labels...)
	})

	t.Run("malformed counter labels skipped", func(t *testing.T) {
		mp := NewLowCardinalityWrappedProvider(testmetrics.NewProvider(t), []string{testBaseName})
		defer mp.Stop()

		c := mp.NewCounter(testCounterName)
		c.Add(1)

		var labels []string

		checkCounter(t, mp, testCounterName, 1, labels...)

		// Odd number of entries. Malformed.
		labels = append(labels, testBaseName, testLabelName, testBaseName, testLabelName, testBaseName)
		c = c.With(labels...)

		c.Add(1)

		checkCounter(t, mp, testCounterName, 2)
	})

	t.Run("malformed gauge labels skipped", func(t *testing.T) {
		mp := NewLowCardinalityWrappedProvider(testmetrics.NewProvider(t), []string{testBaseName})
		defer mp.Stop()

		g := mp.NewGauge(testGaugeName)
		g.Add(1)

		var labels []string

		checkGauge(t, mp, testGaugeName, 1, labels...)

		// Odd number of entries. Malformed.
		labels = append(labels, testBaseName, testLabelName, testBaseName, testLabelName, testBaseName)
		g = g.With(labels...)

		g.Add(1)

		checkGauge(t, mp, testGaugeName, 2)
	})

	t.Run("malformed histogram labels skipped", func(t *testing.T) {
		mp := NewLowCardinalityWrappedProvider(testmetrics.NewProvider(t), []string{testBaseName})
		defer mp.Stop()

		h := mp.NewHistogram(testHistogramName, 50)
		h.Observe(float64(1))

		var labels []string
		var obs = []float64{1}

		checkHistogram(t, mp, testHistogramName, obs, labels...)

		// Odd number of entries. Malformed.
		labels = append(labels, testBaseName, testLabelName, testBaseName, testLabelName, testBaseName)
		h = h.With(labels...)

		h.Observe(float64(1))
		obs = append(obs, float64(1))

		checkHistogram(t, mp, testHistogramName, obs)
	})

	t.Run("malformed explicit histogram labels skipped", func(t *testing.T) {
		mp := NewLowCardinalityWrappedProvider(testmetrics.NewProvider(t), []string{testBaseName})
		defer mp.Stop()

		h := mp.NewExplicitHistogram(testHistogramName, xmetrics.TenSecondDistribution)
		h.Observe(float64(1))

		var labels []string
		var obs = []float64{1}

		checkHistogram(t, mp, testHistogramName, obs, labels...)

		// Odd number of entries. Malformed.
		labels = append(labels, testBaseName, testLabelName, testBaseName, testLabelName, testBaseName)
		h = h.With(labels...)

		h.Observe(float64(1))
		obs = append(obs, float64(1))

		checkHistogram(t, mp, testHistogramName, obs)
	})
}

//nolint:unparam
func checkCounter(t *testing.T, mp xmetrics.Provider, name string, v float64, labelValues ...string) {
	if lcp, ok := mp.(lowCardinalityWrappedProvider); ok {
		if tp, ok := lcp.Provider.(*testmetrics.Provider); ok {
			tp.CheckCounter(name, v, labelValues...)
			return
		}
	}
	t.Error("failed to check counter; could not cast to *testmetrics.Provider")
}

//nolint:unparam
func checkGauge(t *testing.T, mp xmetrics.Provider, name string, v float64, labelValues ...string) {
	if lcp, ok := mp.(lowCardinalityWrappedProvider); ok {
		if tp, ok := lcp.Provider.(*testmetrics.Provider); ok {
			tp.CheckGauge(name, v, labelValues...)
			return
		}
	}
	t.Error("failed to check gauge; could not cast to *testmetrics.Provider")
}

func checkHistogram(t *testing.T, mp xmetrics.Provider, name string, v []float64, labelValues ...string) {
	if lcp, ok := mp.(lowCardinalityWrappedProvider); ok {
		if tp, ok := lcp.Provider.(*testmetrics.Provider); ok {
			tp.CheckObservations(name, v, labelValues...)
			return
		}
	}
	t.Error("failed to check histogram; could not cast to *testmetrics.Provider")
}
