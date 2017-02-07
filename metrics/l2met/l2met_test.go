package l2met

import (
	"bytes"
	"regexp"
	"strconv"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/go-kit/kit/metrics/teststat"
)

func TestCounter(t *testing.T) {
	prefix, name := "abc.", "def"
	label, value := "label", "value" // ignored for l2met
	regex := `^count#` + prefix + name + `=([0-9\.]+)[0-9]+$`
	g := New(prefix, logrus.New())
	counter := g.NewCounter(name).With(label, value)
	valuef := teststat.SumLines(g, regex)
	if err := teststat.TestCounter(counter, valuef); err != nil {
		t.Fatal(err)
	}
}

func TestGauge(t *testing.T) {
	prefix, name := "ghi.", "jkl"
	label, value := "xyz", "abc" // ignored for l2met
	regex := `^sample#` + prefix + name + `=([0-9\.]+)[0-9]+$`
	g := New(prefix, logrus.New())
	gauge := g.NewGauge(name).With(label, value)
	valuef := teststat.LastLine(g, regex)
	if err := teststat.TestGauge(gauge, valuef); err != nil {
		t.Fatal(err)
	}
}

func TestHistogram(t *testing.T) {
	// The histogram test is actually like 4 gauge tests.
	prefix, name := "l2met.", "histogram_test"
	label, value := "abc", "def" // ignored for l2met
	re50 := regexp.MustCompile(`sample#` + prefix + name + `.perc50=([0-9\.]+)[0-9]+`)
	re90 := regexp.MustCompile(`sample#` + prefix + name + `.perc90=([0-9\.]+)[0-9]+`)
	re95 := regexp.MustCompile(`sample#` + prefix + name + `.perc95=([0-9\.]+)[0-9]+`)
	re99 := regexp.MustCompile(`sample#` + prefix + name + `.perc99=([0-9\.]+)[0-9]+`)
	g := New(prefix, logrus.New())
	histogram := g.NewHistogram(name, 50).With(label, value)
	quantiles := func() (float64, float64, float64, float64) {
		var buf bytes.Buffer
		g.WriteTo(&buf)
		match50 := re50.FindStringSubmatch(buf.String())
		p50, _ := strconv.ParseFloat(match50[1], 64)
		match90 := re90.FindStringSubmatch(buf.String())
		p90, _ := strconv.ParseFloat(match90[1], 64)
		match95 := re95.FindStringSubmatch(buf.String())
		p95, _ := strconv.ParseFloat(match95[1], 64)
		match99 := re99.FindStringSubmatch(buf.String())
		p99, _ := strconv.ParseFloat(match99[1], 64)
		return p50, p90, p95, p99
	}
	if err := teststat.TestHistogram(histogram, quantiles, 0.01); err != nil {
		t.Fatal(err)
	}
}

func TestFormatFloat(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{1.0, "1.000000000"},
		{12345678.9, "12345678.900000000"},
		{0.000001, "0.000001000"},
		{0.000000001, "0.000000001"},
	}

	for _, tt := range cases {
		have := formatFloat(tt.in)
		if have != tt.want {
			t.Fatalf("have %s, want %s", have, tt.want)
		}
	}
}
