package librato

import (
	"time"
)

// the json marshalers for the histograms 4 different gauges
func (p *Provider) histogramMeasures(h *Histogram, period int) []measurement {
	h.mu.Lock()
	if h.count == 0 {
		h.mu.Unlock()
		return nil
	}
	count := h.count
	sum := h.sum
	min := h.min
	max := h.max
	sumsq := h.sumsq
	stddev := stddev(sum, sumsq, count)
	last := h.last
	name := h.metricName()
	ts := p.now().Truncate(time.Second * time.Duration(period))

	var attrs map[string]interface{}
	if p.ssa {
		attrs = map[string]interface{}{"aggregate": true}
	}

	percs := []struct {
		n string
		v float64
	}{
		{name + h.percentilePrefix + "99", h.h.Quantile(.99)},
		{name + h.percentilePrefix + "95", h.h.Quantile(.95)},
		{name + h.percentilePrefix + "50", h.h.Quantile(.50)},
	}
	h.reset()
	h.mu.Unlock()

	m := make([]measurement, 0, 4)
	m = append(m, measurement{
		Name:       name,
		Period:     period,
		Time:       ts.Unix(),
		Count:      count,
		Sum:        sum,
		SumSq:      sumsq,
		Min:        min,
		Max:        max,
		Last:       last,
		StdDev:     stddev,
		Attributes: attrs,
		Tags:       p.tagsFor(h.labelValues...),
	})

	for _, perc := range percs {
		m = append(m, measurement{
			Name:       perc.n,
			Period:     period,
			Time:       ts.Unix(),
			Count:      1,
			Sum:        perc.v,
			SumSq:      perc.v * perc.v,
			Min:        perc.v,
			Max:        perc.v,
			Last:       perc.v,
			StdDev:     0,
			Attributes: attrs,
			Tags:       p.tagsFor(h.labelValues...),
		})
	}

	return m
}

// sample the metrics
func (p *Provider) sample(period int) []measurement {
	p.mu.Lock()
	defer p.mu.Unlock() // should only block New{Histogram,Counter,Gauge,Cardinalityounter}

	// TODO: also add cardinality counters.
	if len(p.counters) == 0 && len(p.histograms) == 0 && len(p.gauges) == 0 && len(p.cardinalityCounters) == 0 {
		return nil
	}

	var attrs map[string]interface{}
	if p.ssa {
		attrs = map[string]interface{}{"aggregate": true}
	}

	ts := p.now().Truncate(time.Second * time.Duration(period))

	// Assemble all the data we have to send
	var measurements []measurement
	for _, c := range p.counters {
		var v float64
		if p.resetCounters {
			v = c.ValueReset()
		} else {
			v = c.Value()
		}

		measurements = append(measurements, measurement{
			Name:       c.metricName(),
			Time:       ts.Unix(),
			Period:     period,
			Count:      1,
			Sum:        v,
			SumSq:      v * v,
			Min:        v,
			Max:        v,
			Last:       v,
			StdDev:     0,
			Tags:       p.tagsFor(c.LabelValues()...),
			Attributes: attrs,
		})
	}

	for _, g := range p.gauges {
		v := g.Value()
		measurements = append(measurements, measurement{
			Name:       g.metricName(),
			Time:       ts.Unix(),
			Period:     period,
			Count:      1,
			Sum:        v,
			SumSq:      v * v,
			Min:        v,
			Max:        v,
			Last:       v,
			StdDev:     0,
			Tags:       p.tagsFor(g.LabelValues()...),
			Attributes: attrs,
		})
	}

	for _, h := range p.histograms {
		measurements = append(measurements, p.histogramMeasures(h, period)...)
	}

	for _, c := range p.cardinalityCounters {
		var v float64
		if p.resetCounters {
			v = float64(c.EstimateReset())
		} else {
			v = float64(c.Estimate())
		}
		measurements = append(measurements, measurement{
			Name:       c.metricName(),
			Time:       ts.Unix(),
			Period:     period,
			Count:      1,
			Sum:        v,
			SumSq:      v * v,
			Min:        v,
			Max:        v,
			Last:       v,
			StdDev:     0,
			Tags:       p.tagsFor(c.LabelValues()...),
			Attributes: attrs,
		})
	}

	if p.measurements != nil {
		p.measurements.Set(float64(len(measurements)))
	}

	return measurements
}

func (p *Provider) tagsFor(labelValues ...string) map[string]string {
	if len(labelValues) == 0 {
		return map[string]string{"source": p.source}
	}
	return labelValuesToTags(labelValues...)
}
