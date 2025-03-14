/*
Copyright 2021 Pyrra Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"fmt"
	"strconv"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/pyrra-dev/pyrra/slo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func init() {
	SchemeBuilder.Register(&ServiceLevelObjective{}, &ServiceLevelObjectiveList{})
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:object:root=true

// ServiceLevelObjectiveList contains a list of ServiceLevelObjective
type ServiceLevelObjectiveList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceLevelObjective `json:"items"`
}

// +kubebuilder:object:root=true

// ServiceLevelObjective is the Schema for the ServiceLevelObjectives API
type ServiceLevelObjective struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceLevelObjectiveSpec   `json:"spec,omitempty"`
	Status ServiceLevelObjectiveStatus `json:"status,omitempty"`
}

// ServiceLevelObjectiveSpec defines the desired state of ServiceLevelObjective
type ServiceLevelObjectiveSpec struct {
	// +optional
	// Description describes the ServiceLevelObjective in more detail and
	// gives extra context for engineers that might not directly work on the service.
	Description string `json:"description"`

	// Target is a string that's casted to a float64 between 0 - 100.
	// It represents the desired availability of the service in the given window.
	// float64 are not supported: https://github.com/kubernetes-sigs/controller-tools/issues/245
	Target string `json:"target"`

	// Window within which the Target is supposed to be kept. Usually something like 1d, 7d or 28d.
	Window model.Duration `json:"window"`

	// ServiceLevelIndicator is the underlying data source that indicates how the service is doing.
	// This will be a Prometheus metric with specific selectors for your service.
	ServiceLevelIndicator ServiceLevelIndicator `json:"indicator"`
}

// ServiceLevelIndicator defines the underlying indicator that is a Prometheus metric.
type ServiceLevelIndicator struct {
	// +optional
	// Ratio is the indicator that measures against errors / total events.
	Ratio *RatioIndicator `json:"ratio,omitempty"`

	// +optional
	// Latency is the indicator that measures a certain percentage to be fast than.
	Latency *LatencyIndicator `json:"latency,omitempty"`
}

type RatioIndicator struct {
	// Errors is the metric that returns how many errors there are.
	Errors Query `json:"errors"`
	// Total is the metric that returns how many requests there are in total.
	Total Query `json:"total"`
}

type LatencyIndicator struct {
	// Success is the metric that returns how many errors there are.
	Success Query `json:"success"`
	// Total is the metric that returns how many requests there are in total.
	Total Query `json:"total"`
}

// Query contains a PromQL metric
type Query struct {
	Metric string `json:"metric"`
}

// ServiceLevelObjectiveStatus defines the observed state of ServiceLevelObjective
type ServiceLevelObjectiveStatus struct{}

func (in ServiceLevelObjective) Internal() (slo.Objective, error) {
	target, err := strconv.ParseFloat(in.Spec.Target, 64)
	if err != nil {
		return slo.Objective{}, err
	}

	if in.Spec.ServiceLevelIndicator.Ratio != nil && in.Spec.ServiceLevelIndicator.Latency != nil {
		return slo.Objective{}, fmt.Errorf("cannot have ratio and latency indicators at the same time")
	}

	var ratio *slo.RatioIndicator
	if in.Spec.ServiceLevelIndicator.Ratio != nil {
		totalExpr, err := parser.ParseExpr(in.Spec.ServiceLevelIndicator.Ratio.Total.Metric)
		if err != nil {
			return slo.Objective{}, err
		}

		totalVec, ok := totalExpr.(*parser.VectorSelector)
		if !ok {
			return slo.Objective{}, fmt.Errorf("ratio total metric is not a VectorSelector")
		}

		errorExpr, err := parser.ParseExpr(in.Spec.ServiceLevelIndicator.Ratio.Errors.Metric)
		if err != nil {
			return slo.Objective{}, err
		}

		errorVec, ok := errorExpr.(*parser.VectorSelector)
		if !ok {
			return slo.Objective{}, fmt.Errorf("ratio error metric is not a VectorSelector")
		}

		// Copy the matchers to get rid of the re field for unit testing...
		errorMatchers := make([]*labels.Matcher, len(errorVec.LabelMatchers))
		for i, matcher := range errorVec.LabelMatchers {
			errorMatchers[i] = &labels.Matcher{Type: matcher.Type, Name: matcher.Name, Value: matcher.Value}
		}

		ratio = &slo.RatioIndicator{
			Errors: slo.Metric{
				Name:          errorVec.Name,
				LabelMatchers: errorMatchers,
			},
			Total: slo.Metric{
				Name:          totalVec.Name,
				LabelMatchers: totalVec.LabelMatchers,
			},
		}
	}

	var latency *slo.LatencyIndicator
	if in.Spec.ServiceLevelIndicator.Latency != nil {
		totalExpr, err := parser.ParseExpr(in.Spec.ServiceLevelIndicator.Latency.Total.Metric)
		if err != nil {
			return slo.Objective{}, err
		}

		totalVec, ok := totalExpr.(*parser.VectorSelector)
		if !ok {
			return slo.Objective{}, fmt.Errorf("latency total metric is not a VectorSelector")
		}

		// Copy the matchers to get rid of the re field for unit testing...
		totalMatchers := make([]*labels.Matcher, len(totalVec.LabelMatchers))
		for i, matcher := range totalVec.LabelMatchers {
			totalMatchers[i] = &labels.Matcher{Type: matcher.Type, Name: matcher.Name, Value: matcher.Value}
		}

		successExpr, err := parser.ParseExpr(in.Spec.ServiceLevelIndicator.Latency.Success.Metric)
		if err != nil {
			return slo.Objective{}, err
		}

		successVec, ok := successExpr.(*parser.VectorSelector)
		if !ok {
			return slo.Objective{}, fmt.Errorf("latency success metric is not a VectorSelector")
		}

		// Copy the matchers to get rid of the re field for unit testing...
		successMatchers := make([]*labels.Matcher, len(successVec.LabelMatchers))
		for i, matcher := range successVec.LabelMatchers {
			successMatchers[i] = &labels.Matcher{Type: matcher.Type, Name: matcher.Name, Value: matcher.Value}
		}

		latency = &slo.LatencyIndicator{
			Success: slo.Metric{
				Name:          successVec.Name,
				LabelMatchers: successMatchers,
			},
			Total: slo.Metric{
				Name:          totalVec.Name,
				LabelMatchers: totalMatchers,
			},
		}
	}

	inCopy := in.DeepCopy()
	inCopy.ManagedFields = nil
	delete(inCopy.Annotations, "kubectl.kubernetes.io/last-applied-configuration")

	config, err := yaml.Marshal(inCopy)
	if err != nil {
		return slo.Objective{}, fmt.Errorf("failed to marshal resource as config")
	}

	return slo.Objective{
		Name:        in.GetName(),
		Namespace:   in.GetNamespace(),
		Description: in.Spec.Description,
		Target:      target / 100,
		Window:      in.Spec.Window,
		Config:      string(config),
		Indicator: slo.Indicator{
			Ratio:   ratio,
			Latency: latency,
		},
	}, nil
}
