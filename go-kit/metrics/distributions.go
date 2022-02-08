package metrics

var (
	// FiveSecondDistribution is a percentile distribution between 0 and 5000 milliseconds.
	//
	// Generated Distribution
	//
	//     []float64{10, 55, 255, 505, 1255, 2505, 3755, 4505, 4755, 4955, 5000}
	FiveSecondDistribution = WithStandardPercentiles(0, 5000)

	// ThirtySecondDistribution is a percentile distribution between 0 and 30,000 milliseconds.
	//
	// Generated Distribution
	//
	//     []float64{60, 330, 1530, 3030, 7530, 15030, 22530, 27030, 28530, 29730, 30000}
	ThirtySecondDistribution = WithStandardPercentiles(0, 30000)

	standardPercentiles = []float64{0.001, 0.01, 0.05, 0.1, 0.25, 0.5, 0.75, 0.9, 0.95, 0.99, 0.999}
)

type (
	// DistributionFunc is able to return an explicit boundaries slice for a given distribution.
	DistributionFunc func() []float64
)

// WithStandardPercentiles returns a bulls horn shaped distribution where the
// lower and upper boundaries are captured using more precision than the
// middle. This distribution is most useful for collecting histograms where you
// are most interested in the upper and/or lower boundaries like the P95, P99,
// P999 or P05, P01, P001.
//
// This standard percentile distribution used here will generate boundaries
// with the following percentile distribution
// P(0.001), P(0.01), P(0.05), P(0.1), P(0.25), P(0.5), P(0.75), P(0.9), P(0.95), P(0.99), P(0.999)
func WithStandardPercentiles(min, max float64) DistributionFunc {
	return WithPercentileDistribution(min, max, standardPercentiles)
}

// WithPercentileDistribution will generate boundaries by scaling the values
// between the min and the max according to supplied percentile distribution
// pattern.
func WithPercentileDistribution(min, max float64, pattern []float64) DistributionFunc {
	return func() []float64 {
		boundaries := make([]float64, len(pattern))

		s := min + max
		l := s * pattern[len(pattern)-1]

		for i, p := range pattern {
			boundaries[i] = (s * p) + (s - l)
		}

		return boundaries
	}
}
