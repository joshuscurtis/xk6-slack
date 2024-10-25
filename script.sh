#!/bin/bash

# Exit on any error
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_status() {
    echo -e "${BLUE}➡️ $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go first."
    exit 1
fi

# Get GitHub username
read -p "Enter your GitHub username: " GITHUB_USERNAME

# Create root directory
print_status "Creating project structure..."
mkdir -p xk6-slack/{.github/workflows,examples}
cd xk6-slack

# Initialize git
print_status "Initializing git repository..."
git init

# Initialize Go module
print_status "Initializing Go module..."
go mod init github.com/$GITHUB_USERNAME/xk6-slack
go get github.com/slack-go/slack
go get go.k6.io/k6@latest

# Create main Go file at root level
print_status "Creating Go source files..."

# Create slack.go at root level
cat > slack.go << 'EOL'
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
EOL

# Create test file
cat > slack_test.go << 'EOL'
package slack

import (
    "testing"
)

func TestClientConfiguration(t *testing.T) {
    client := &Client{}
    
    // Test empty token
    err := client.Configure("", Config{}, "test-user")
    if err == nil {
        t.Error("Expected error with empty token, got none")
    }
    
    // Test valid configuration
    err = client.Configure("test-token", Config{
        SlackChannelID: "test-channel",
    }, "test-user")
    if err != nil {
        t.Errorf("Expected no error, got %v", err)
    }
}
EOL

# Create .gitignore
cat > .gitignore << 'EOL'
# Binaries
k6
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with `go test -c`
*.test

# Output of the go coverage tool
*.out

# Dependency directories
vendor/

# IDE specific files
.idea/
.vscode/
*.swp
*.swo

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db
EOL

# Create Makefile
cat > Makefile << 'EOL'
SHELL := /bin/bash

.PHONY: test
test:
	go test -v -race ./...

.PHONY: build
build:
	xk6 build --with github.com/$(shell git config --get remote.origin.url | sed 's/.*:\(.*\)\.git/\1/')@latest

.PHONY: lint
lint:
	golangci-lint run

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: setup
setup:
	go mod download
	go install github.com/golangci/golint/cmd/golint@latest
	go install go.k6.io/xk6/cmd/xk6@latest

.PHONY: clean
clean:
	rm -f k6
	go clean -cache
EOL

# Create GitHub Actions workflow
mkdir -p .github/workflows
cat > .github/workflows/test.yml << 'EOL'
name: Test

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.19'

    - name: Install dependencies
      run: |
        go mod download
        go install go.k6.io/xk6/cmd/xk6@latest

    - name: Run tests
      run: go test -v -race ./...

    - name: Build
      run: xk6 build --with github.com/${{ github.repository }}@latest
EOL

# Create example script
mkdir -p examples
cat > examples/basic.js << 'EOL'
import slack from 'k6/x/slack';

export const options = {
    vus: 1,
    duration: '30s',
};

const SLACK_TOKEN = __ENV.SLACK_TOKEN;
const SLACK_CHANNEL = __ENV.SLACK_CHANNEL;
const USER = __ENV.GITLAB_USER_LOGIN || 'k6-user';

const slackConfig = {
    slackChannelID: SLACK_CHANNEL,
    dashboardUrls: {
        'K6 Dashboard': '/path-to/k6-dashboard',
    },
    graphUrls: {
        'Response Time Trend': 'https://your-graph-url',
    },
};

const slackClient = new slack.Client();

export function setup() {
    slackClient.configure(SLACK_TOKEN, slackConfig, USER);
    slackClient.sendMessage('Start');
}

export function handleSummary(data) {
    const metrics = {
        'Average Response Time': `${data.metrics.http_req_duration.values.avg.toFixed(2)}ms`,
        'P95 Response Time': `${data.metrics.http_req_duration.values.p95.toFixed(2)}ms`,
        'Request Rate': `${data.metrics.iterations.values.rate.toFixed(2)}/s`,
    };
    
    slackClient.addTestMetrics(metrics);
    slackClient.sendMessage('End');
    return {};
}
EOL

# Create README.md
cat > README.md << EOL
# xk6-slack

xk6 extension for sending k6 test results to Slack.

## Build

1. Install xk6:
\`\`\`bash
go install go.k6.io/xk6/cmd/xk6@latest
\`\`\`

2. Build k6 with the extension:
\`\`\`bash
xk6 build --with github.com/$GITHUB_USERNAME/xk6-slack@latest
\`\`\`

## Usage

\`\`\`javascript
import slack from 'k6/x/slack';

const slackConfig = {
    slackChannelID: __ENV.SLACK_CHANNEL,
    dashboardUrls: {
        'K6 Dashboard': '/your/dashboard/url',
    },
};

export function setup() {
    const slackClient = new slack.Client();
    slackClient.configure(__ENV.SLACK_TOKEN, slackConfig, __ENV.GITLAB_USER_LOGIN);
    slackClient.sendMessage('Start');
}
\`\`\`

## Environment Variables

- SLACK_TOKEN: Slack Bot User OAuth Token
- SLACK_CHANNEL: Channel ID to post messages
- GITLAB_USER_LOGIN: Username to attribute the test run

## Development

1. Clone the repo
2. Install dependencies: \`make setup\`
3. Run tests: \`make test\`
4. Build locally: \`make build\`
EOL

# Initialize git repository
print_status "Initializing git repository..."
git add .
git commit -m "Initial commit"

print_status "Setting up development tools..."
make setup

print_success "Repository setup complete! Next steps:"
echo -e "${GREEN}"
echo "1. Create a new repository on GitHub: https://github.com/new"
echo "2. Push your code:"
echo "   git remote add origin git@github.com:$GITHUB_USERNAME/xk6-slack.git"
echo "   git branch -M main"
echo "   git push -u origin main"
echo -e "${NC}"

The key changes are:
1. Moved the main code from `pkg/slack/client.go` to `slack.go` in the root directory
2. Added `slack_test.go` in the root directory
3. Updated the directory structure to match xk6 expectations
4. Added proper imports for k6 packages

The new structure will be:

```
xk6-slack/
├── .github/
│   └── workflows/
│       └── test.yml
├── examples/
│   └── basic.js
├── slack.go
├── slack_test.go
├── .gitignore
├──
