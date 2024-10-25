package slack

import (
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

func (*SlackAPI) NewModuleInstance(m modules.VU) modules.Instance {
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
