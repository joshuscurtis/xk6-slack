package slack

import (
	"encoding/json"
	"testing"
)

// TestData represents a complete k6 test data structure
type TestData struct {
	Metrics map[string]MetricData `json:"metrics"`
}

type MetricData struct {
	Contains string                 `json:"contains,omitempty"`
	Values   map[string]interface{} `json:"values"`
	Type     string                 `json:"type,omitempty"`
}

func TestSendTestResults(t *testing.T) {
	// Sample metrics data
	sampleMetrics := `{
		"http_req_duration": {
			"contains": "time",
			"values": {
				"avg": 76.2412846153846,
				"min": 64.127,
				"med": 73.7895,
				"max": 139.283,
				"p(90)": 82.8032,
				"p(95)": 87.90965
			},
			"type": "trend"
		},
		"http_reqs": {
			"type": "counter",
			"contains": "default",
			"values": {
				"count": 130,
				"rate": 12.642716822593506
			}
		},
		"http_req_failed": {
			"type": "rate",
			"contains": "default",
			"values": {
				"passes": 0,
				"fails": 130,
				"rate": 0
			}
		},
		"data_received": {
			"contains": "data",
			"values": {
				"count": 2890872,
				"rate": 281142.1235874195
			},
			"type": "counter"
		},
		"vus": {
			"type": "gauge",
			"contains": "default",
			"values": {
				"value": 1,
				"min": 1,
				"max": 1
			}
		}
	}`

	// Verify the sample data is valid JSON
	var testData TestData
	err := json.Unmarshal([]byte(sampleMetrics), &testData)
	if err != nil {
		t.Fatalf("Invalid test metrics JSON: %v", err)
	}

	// Create test cases
	tests := []struct {
		name        string
		metricsJSON string
		token       string
		channel     string
		expectError bool
	}{
		{
			name:        "Valid metrics data",
			metricsJSON: sampleMetrics,
			token:       "test-token",
			channel:     "test-channel",
			expectError: false,
		},
		{
			name:        "Invalid JSON",
			metricsJSON: "invalid json",
			token:       "test-token",
			channel:     "test-channel",
			expectError: true,
		},
		{
			name:        "Empty metrics",
			metricsJSON: "{}",
			token:       "test-token",
			channel:     "test-channel",
			expectError: false,
		},
		{
			name:        "Unconfigured client",
			metricsJSON: sampleMetrics,
			token:       "",
			channel:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create and configure client
			client := &Client{}
			if tt.token != "" && tt.channel != "" {
				err := client.Configure(tt.token, tt.channel)
				if err != nil {
					t.Fatalf("Failed to configure client: %v", err)
				}
			}

			// Test SendTestResults
			err := client.SendTestResults(tt.metricsJSON)

			// Check if error matches expectation
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// Test metric value extraction
func TestGetSafeMetricValue(t *testing.T) {
	sampleMetrics := map[string]struct{ Values map[string]interface{} }{
		"http_req_duration": {
			Values: map[string]interface{}{
				"avg":   76.24,
				"p(95)": 87.91,
			},
		},
	}

	tests := []struct {
		name       string
		metricName string
		valuePath  string
		want       interface{}
	}{
		{
			name:       "Valid metric avg",
			metricName: "http_req_duration",
			valuePath:  "avg",
			want:       76.24,
		},
		{
			name:       "Valid metric p95",
			metricName: "http_req_duration",
			valuePath:  "p(95)",
			want:       87.91,
		},
		{
			name:       "Non-existent metric",
			metricName: "non_existent",
			valuePath:  "avg",
			want:       nil,
		},
		{
			name:       "Non-existent value",
			metricName: "http_req_duration",
			valuePath:  "non_existent",
			want:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSafeMetricValue(sampleMetrics, tt.metricName, tt.valuePath)
			if got != tt.want {
				t.Errorf("getSafeMetricValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test value formatting
func TestFormatValue(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		unit  string
		want  string
	}{
		{
			name:  "Format milliseconds",
			value: 76.24,
			unit:  "ms",
			want:  "76.24ms",
		},
		{
			name:  "Format rate",
			value: 12.64,
			unit:  "rate",
			want:  "12.64/s",
		},
		{
			name:  "Format percentage",
			value: 0.95,
			unit:  "percent",
			want:  "95.00%",
		},
		{
			name:  "Format bytes (KB)",
			value: 1024.0,
			unit:  "bytes",
			want:  "1.00 KB",
		},
		{
			name:  "Format bytes (MB)",
			value: 2890872.0,
			unit:  "bytes",
			want:  "2.76 MB",
		},
		{
			name:  "Format nil value",
			value: nil,
			unit:  "ms",
			want:  "N/A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatValue(tt.value, tt.unit)
			if got != tt.want {
				t.Errorf("formatValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
