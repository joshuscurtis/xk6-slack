package slack

import (
	"encoding/json"
	"fmt"
	"github.com/slack-go/slack"
	"go.k6.io/k6/js/modules"
)

func init() {
	modules.Register("k6/x/slack", new(SlackAPI))
}

type SlackAPI struct{}

type Client struct {
	api     *slack.Client
	channel string
}

// Exports maps the methods that will be available in JavaScript
func (c *Client) Exports() modules.Exports {
	return modules.Exports{
		Named: map[string]interface{}{
			"configure":       c.Configure,
			"sendMessage":     c.SendMessage,
			"sendTestResults": c.SendTestResults,
		},
	}
}

func (*SlackAPI) NewModuleInstance(_ modules.VU) modules.Instance {
	return &Client{}
}

// Configure sets up the slack client
func (c *Client) Configure(token string, channel string) error {
	if token == "" {
		return fmt.Errorf("slack token cannot be empty")
	}
	if channel == "" {
		return fmt.Errorf("slack channel cannot be empty")
	}
	c.api = slack.New(token)
	c.channel = channel
	return nil
}

// SendMessage sends a simple message to Slack
func (c *Client) SendMessage(message string) error {
	if c.api == nil {
		return fmt.Errorf("slack client not configured, call Configure() first")
	}
	_, _, err := c.api.PostMessage(
		c.channel,
		slack.MsgOptionText(message, false),
	)
	return err
}

// K6Metric represents a single k6 metric
type K6Metric struct {
	Contains string                 `json:"contains"`
	Values   map[string]interface{} `json:"values"`
	Type     string                 `json:"type"`
}

// K6Data represents the complete k6 test data structure
type K6Data struct {
	Metrics   map[string]K6Metric `json:"metrics"`
	RootGroup struct {
		Checks []struct {
			Name   string `json:"name"`
			Passes int64  `json:"passes"`
			Fails  int64  `json:"fails"`
		} `json:"checks"`
	} `json:"root_group"`
}

// MetricResults stores processed metric values
type MetricResults struct {
	Status      string
	TestName    string
	Environment string
	Metrics     map[string]string
	Checks      map[string]CheckResult
}

// CheckResult represents the result of a single check
type CheckResult struct {
	Passes int64
	Fails  int64
	Rate   string
}

// formatNumber formats numerical values with 2 decimal places
func formatNumber(num float64) string {
	return fmt.Sprintf("%.2f", num)
}

// formatDuration formats duration values with ms suffix
func formatDuration(value float64) string {
	return fmt.Sprintf("%.2fms", value)
}

// SendTestResults processes and sends formatted test results to Slack
func (c *Client) SendTestResults(data interface{}) error {
	if c.api == nil {
		return fmt.Errorf("slack client not configured, call Configure() first")
	}

	// Marshal and unmarshal to ensure we have the correct data structure
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling data: %v", err)
	}

	var k6Data K6Data
	if err := json.Unmarshal(jsonData, &k6Data); err != nil {
		return fmt.Errorf("error unmarshaling data: %v", err)
	}

	results := MetricResults{
		Status:      "passed",
		TestName:    "API Performance Test",
		Environment: "staging",
		Metrics:     make(map[string]string),
		Checks:      make(map[string]CheckResult),
	}

	// Helper function to safely get metric values
	getMetricValue := func(metricName, valueName string) float64 {
		if metric, ok := k6Data.Metrics[metricName]; ok {
			if value, ok := metric.Values[valueName].(float64); ok {
				return value
			}
		}
		return 0
	}

	// Process HTTP duration metrics
	results.Metrics["Response Time (avg)"] = formatDuration(getMetricValue("http_req_duration", "avg"))
	results.Metrics["Response Time (min)"] = formatDuration(getMetricValue("http_req_duration", "min"))
	results.Metrics["Response Time (med)"] = formatDuration(getMetricValue("http_req_duration", "med"))
	results.Metrics["Response Time (max)"] = formatDuration(getMetricValue("http_req_duration", "max"))
	results.Metrics["Response Time (p90)"] = formatDuration(getMetricValue("http_req_duration", "p(90)"))
	results.Metrics["Response Time (p95)"] = formatDuration(getMetricValue("http_req_duration", "p(95)"))

	// Process other timing metrics
	results.Metrics["Time to First Byte (avg)"] = formatDuration(getMetricValue("http_req_waiting", "avg"))
	results.Metrics["Connection Time (avg)"] = formatDuration(getMetricValue("http_req_connecting", "avg"))
	results.Metrics["TLS Handshake (avg)"] = formatDuration(getMetricValue("http_req_tls_handshaking", "avg"))
	results.Metrics["Sending Time (avg)"] = formatDuration(getMetricValue("http_req_sending", "avg"))
	results.Metrics["Receiving Time (avg)"] = formatDuration(getMetricValue("http_req_receiving", "avg"))
	results.Metrics["Blocking Time (avg)"] = formatDuration(getMetricValue("http_req_blocked", "avg"))

	// Process throughput metrics
	results.Metrics["Data Received"] = fmt.Sprintf("%.2f KB/s", getMetricValue("data_received", "rate")/1024)
	results.Metrics["Data Sent"] = fmt.Sprintf("%.2f KB/s", getMetricValue("data_sent", "rate")/1024)

	// Process request metrics
	results.Metrics["Total Requests"] = fmt.Sprintf("%.0f", getMetricValue("http_reqs", "count"))
	results.Metrics["Request Rate"] = fmt.Sprintf("%.2f/s", getMetricValue("http_reqs", "rate"))

	// Process iteration metrics
	results.Metrics["Iterations"] = fmt.Sprintf("%.0f", getMetricValue("iterations", "count"))
	results.Metrics["Iteration Rate"] = fmt.Sprintf("%.2f/s", getMetricValue("iterations", "rate"))

	// Process VU metrics
	results.Metrics["Virtual Users"] = fmt.Sprintf("%.0f", getMetricValue("vus", "value"))

	// Process success rate
	failRate := getMetricValue("http_req_failed", "rate")
	if failRate > 0 {
		results.Status = "failed"
	}
	results.Metrics["Success Rate"] = fmt.Sprintf("%.2f%%", 100-(failRate*100))

	// Process checks
	for _, check := range k6Data.RootGroup.Checks {
		total := float64(check.Passes + check.Fails)
		rate := "100%"
		if total > 0 {
			rate = fmt.Sprintf("%.2f%%", (float64(check.Passes)/total)*100)
		}

		results.Checks[check.Name] = CheckResult{
			Passes: check.Passes,
			Fails:  check.Fails,
			Rate:   rate,
		}
	}

	// Create Slack message blocks
	var blocks []slack.Block

	// Header block
	blocks = append(blocks,
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", fmt.Sprintf("Performance Test Results: %s", results.TestName), true, false),
		),
		slack.NewDividerBlock(),
	)

	// Status and environment blocks
	statusEmoji := "✅"
	if results.Status != "passed" {
		statusEmoji = "❌"
	}

	blocks = append(blocks,
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*Status:* %s %s", results.Status, statusEmoji), false, false),
			nil, nil,
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*Environment:* %s", results.Environment), false, false),
			nil, nil,
		),
		slack.NewDividerBlock(),
	)

	// Metrics blocks
	blocks = append(blocks,
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", "*Test Metrics:*", false, false),
			nil, nil,
		),
	)

	// Add metrics in groups of 10 (Slack's limit for fields in a section)
	var fields []*slack.TextBlockObject
	for name, value := range results.Metrics {
		fields = append(fields,
			slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s*\n%s", name, value), false, false),
		)

		if len(fields) == 10 {
			blocks = append(blocks, slack.NewSectionBlock(nil, fields, nil))
			fields = []*slack.TextBlockObject{}
		}
	}

	// Add remaining metrics
	if len(fields) > 0 {
		blocks = append(blocks, slack.NewSectionBlock(nil, fields, nil))
	}

	// Add checks section if there are any checks
	if len(results.Checks) > 0 {
		blocks = append(blocks,
			slack.NewDividerBlock(),
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", "*Checks Results:*", false, false),
				nil, nil,
			),
		)

		fields = []*slack.TextBlockObject{}
		for name, check := range results.Checks {
			fields = append(fields,
				slack.NewTextBlockObject("mrkdwn",
					fmt.Sprintf("*%s*\n✓ %d | ✗ %d | Rate: %s",
						name, check.Passes, check.Fails, check.Rate),
					false, false,
				),
			)

			if len(fields) == 10 {
				blocks = append(blocks, slack.NewSectionBlock(nil, fields, nil))
				fields = []*slack.TextBlockObject{}
			}
		}

		if len(fields) > 0 {
			blocks = append(blocks, slack.NewSectionBlock(nil, fields, nil))
		}
	}

	// Send message with blocks
	_, _, err = c.api.PostMessage(
		c.channel,
		slack.MsgOptionBlocks(blocks...),
	)
	return err
}
