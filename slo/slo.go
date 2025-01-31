package slo

import (
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

type Objective struct {
	Name        string
	Namespace   string
	Description string
	Target      float64
	Window      model.Duration
	Config      string

	Indicator Indicator
}

type Indicator struct {
	Ratio   *RatioIndicator
	Latency *LatencyIndicator
}

type RatioIndicator struct {
	Errors Metric
	Total  Metric
}

type LatencyIndicator struct {
	Success Metric
	Total   Metric
}

type Metric struct {
	Name          string
	LabelMatchers []*labels.Matcher
}

func (m Metric) Metric() string {
	v := parser.VectorSelector{Name: m.Name, LabelMatchers: m.LabelMatchers}
	return v.String()
}
