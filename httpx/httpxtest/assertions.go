package httpxtest

import (
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// AssertMetricValue asserts that a metric has the expected value.
// This works with Counter and Gauge metrics.
func AssertMetricValue(t *testing.T, collector prometheus.Collector, expected float64) {
	t.Helper()

	metricCh := make(chan prometheus.Metric, 10)
	collector.Collect(metricCh)
	close(metricCh)

	if len(metricCh) == 0 {
		t.Fatalf("no metrics collected")
	}

	metric := <-metricCh
	pb := &dto.Metric{}
	if err := metric.Write(pb); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}

	var actual float64
	if pb.Counter != nil {
		actual = pb.Counter.GetValue()
	} else if pb.Gauge != nil {
		actual = pb.Gauge.GetValue()
	} else {
		t.Fatalf("metric is neither Counter nor Gauge")
	}

	if actual != expected {
		t.Errorf("metric value mismatch: got %v, want %v", actual, expected)
	}
}

// AssertMetricExists asserts that a metric with the given name exists in the registry.
func AssertMetricExists(t *testing.T, registry *prometheus.Registry, metricName string) {
	t.Helper()

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	for _, family := range families {
		if family.GetName() == metricName {
			return // Found it
		}
	}

	t.Errorf("metric %q not found in registry", metricName)
}

// GetMetricValue retrieves the value of a metric with the given name and labels.
// Returns an error if the metric is not found.
func GetMetricValue(registry *prometheus.Registry, metricName string, labels map[string]string) (float64, error) {
	families, err := registry.Gather()
	if err != nil {
		return 0, fmt.Errorf("failed to gather metrics: %w", err)
	}

	for _, family := range families {
		if family.GetName() != metricName {
			continue
		}

		for _, metric := range family.GetMetric() {
			if matchesLabels(metric, labels) {
				if metric.Counter != nil {
					return metric.Counter.GetValue(), nil
				}
				if metric.Gauge != nil {
					return metric.Gauge.GetValue(), nil
				}
				if metric.Histogram != nil {
					return float64(metric.Histogram.GetSampleCount()), nil
				}
			}
		}
	}

	return 0, fmt.Errorf("metric %q with labels %v not found", metricName, labels)
}

// matchesLabels checks if a metric's labels match the expected labels.
func matchesLabels(metric *dto.Metric, expectedLabels map[string]string) bool {
	if len(expectedLabels) == 0 {
		return true // No label filter
	}

	metricLabels := make(map[string]string)
	for _, label := range metric.GetLabel() {
		metricLabels[label.GetName()] = label.GetValue()
	}

	for key, expectedValue := range expectedLabels {
		actualValue, exists := metricLabels[key]
		if !exists || actualValue != expectedValue {
			return false
		}
	}

	return true
}

// AssertMetricValueWithLabels asserts that a metric with specific labels has the expected value.
func AssertMetricValueWithLabels(t *testing.T, registry *prometheus.Registry, metricName string, labels map[string]string, expected float64) {
	t.Helper()

	actual, err := GetMetricValue(registry, metricName, labels)
	if err != nil {
		t.Fatalf("failed to get metric value: %v", err)
	}

	if actual != expected {
		t.Errorf("metric %q with labels %v: got %v, want %v", metricName, labels, actual, expected)
	}
}
