package slack

import (
	"encoding/json"
	"fmt"
	"time"
	"github.com/slack-go/slack"
	"go.k6.io/k6/js/modules"
)

// Config types
type DashboardURLs map[string]string
type GraphURLs map[string]string

type Config struct {
	SlackChannelID string        `json:"slackChannelID"`
	DashboardURLs  DashboardURLs `json:"dashboardUrls"`
	GraphURLs      GraphURLs     `json:"graphUrls"`
}

type MessageType string

const (
	StartMessage MessageType = "Start"
	EndMessage   MessageType = "End"
)

func init() {
	modules.Register("k6/x/slack", new(SlackAPI))
}

type SlackAPI struct{}

type Client struct {
	token           string
	api             *slack.Client
	config          Config
	dashboardStart  time.Time
	executionUser   string
}

func (*SlackAPI) New() interface{} {
	return &Client{}
}

func (c *Client) Configure(token string, config Config, user string) error {
	if token == "" {
		return fmt.Errorf("slack token cannot be empty")
	}
	c.token = token
	c.api = slack.New(token)
	c.config = config
	c.dashboardStart = time.Now()
	c.executionUser = user
	return nil
}

// createDashboardLinks generates dashboard URLs with proper time ranges
func (c *Client) createDashboardLinks(isStart bool) map[string]string {
	links := make(map[string]string)
	endTime := time.Now()
	
	for name, baseURL := range c.config.DashboardURLs {
		timeRange := fmt.Sprintf("from_ts=%d&to_ts=%d", 
			c.dashboardStart.Unix(),
			endTime.Unix())
		
		if isStart {
			// For start message, set end time to start time + 1 hour
			timeRange = fmt.Sprintf("from_ts=%d&to_ts=%d", 
				c.dashboardStart.Unix(),
				c.dashboardStart.Add(1*time.Hour).Unix())
		}
		
		links[name] = fmt.Sprintf("%s?%s", baseURL, timeRange)
	}
	return links
}

// createButtonBlocks creates Slack button blocks for dashboard links
func (c *Client) createButtonBlocks(dashboardURLs map[string]string) []slack.Block {
	var blocks []slack.Block
	
	for name, url := range dashboardURLs {
		blocks = append(blocks,
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", 
					fmt.Sprintf("View the live test execution on the %s dashboard", name),
					false, false),
				nil,
				slack.NewAccessory(
					slack.NewButtonBlockElement(
						"",
						name,
						slack.NewTextBlockObject("plain_text", "View Dashboard :bar_chart:", true, false)).
						WithURL(url),
				),
			),
		)
	}
	
	blocks = append(blocks, slack.NewDividerBlock())
	return blocks
}

// createGraphBlocks creates Slack blocks for graph images
func (c *Client) createGraphBlocks(graphURLs map[string]string) []slack.Block {
	var blocks []slack.Block
	
	blocks = append(blocks,
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", ":stopwatch: Test Results Summary", true, false),
		),
		slack.NewDividerBlock(),
	)
	
	for title, url := range graphURLs {
		blocks = append(blocks,
			slack.NewImageBlock(
				url,
				title,
				"",
				slack.NewTextBlockObject("plain_text", title, false, false),
			),
			slack.NewDividerBlock(),
		)
	}
	
	return blocks
}

// SendMessage sends either a start or end message to Slack
func (c *Client) SendMessage(messageType MessageType) error {
	if c.api == nil {
		return fmt.Errorf("slack client not configured, call Configure() first")
	}

	var blocks []slack.Block
	
	// Header blocks
	blocks = append(blocks,
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", 
				fmt.Sprintf("Performance Test %s", messageType), 
				false, false),
		),
		slack.NewDividerBlock(),
		slack.NewSectionBlock(
			nil,
			[]*slack.TextBlockObject{
				slack.NewTextBlockObject("mrkdwn", 
					fmt.Sprintf("User: %s", c.executionUser), 
					false, false),
			},
			nil,
		),
		slack.NewDividerBlock(),
	)

	// Add appropriate blocks based on message type
	if messageType == StartMessage {
		dashboardLinks := c.createDashboardLinks(true)
		blocks = append(blocks, c.createButtonBlocks(dashboardLinks)...)
	} else if messageType == EndMessage {
		// Add graph blocks
		if len(c.config.GraphURLs) > 0 {
			blocks = append(blocks, c.createGraphBlocks(c.config.GraphURLs)...)
		}
		
		// Add dashboard links with full time range
		dashboardLinks := c.createDashboardLinks(false)
		blocks = append(blocks, c.createButtonBlocks(dashboardLinks)...)
	}

	// Send message
	_, _, err := c.api.PostMessage(
		c.config.SlackChannelID,
		slack.MsgOptionBlocks(blocks...),
	)

	return err
}

// AddTestMetrics adds test metrics to the message (optional)
func (c *Client) AddTestMetrics(metrics map[string]interface{}) error {
	if c.api == nil {
		return fmt.Errorf("slack client not configured, call Configure() first")
	}

	var blocks []slack.Block
	blocks = append(blocks,
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", "Test Metrics", false, false),
		),
		slack.NewDividerBlock(),
	)

	// Create fields for metrics
	var fields []*slack.TextBlockObject
	for name, value := range metrics {
		fields = append(fields,
			slack.NewTextBlockObject("mrkdwn",
				fmt.Sprintf("*%s*\n%v", name, value),
				false, false),
		)
	}

	// Split fields into groups of 10 (Slack's limit)
	for i := 0; i < len(fields); i += 10 {
		end := i + 10
		if end > len(fields) {
			end = len(fields)
		}
		blocks = append(blocks,
			slack.NewSectionBlock(nil, fields[i:end], nil),
		)
	}

	// Send update
	_, _, err := c.api.PostMessage(
		c.config.SlackChannelID,
		slack.MsgOptionBlocks(blocks...),
	)

	return err
}
