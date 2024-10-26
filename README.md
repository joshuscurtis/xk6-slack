# xk6-slack

A k6 extension that enables sending test results and notifications to Slack channels during and after k6 test execution. This extension is perfect for teams who want to monitor their performance tests and receive immediate feedback through Slack.

## Features

- 🚀 Send real-time notifications to Slack during test execution
- 📊 Send formatted test results with detailed metrics
- 🎨 Beautiful Slack message formatting using blocks
- 🔧 Easy configuration and setup
- 📈 Comprehensive metrics reporting including:
  - Response time statistics (avg, min, med, max, p90, p95)
  - Request breakdown metrics (TTFB, connection time, TLS handshake)
  - Volume metrics (request rate, data transfer)
  - Test execution statistics
  - Success rates and check results

## Prerequisites

- Go 1.19 or later
- k6 v0.43.0 or later
- Slack workspace with permission to create bot tokens

## Installation

1. First, install xk6:
```bash
go install go.k6.io/xk6/cmd/xk6@latest
```

2. Build k6 with the slack extension:
```bash
xk6 build --with github.com/joshuscurtis/xk6-slack@latest
```

## Configuration

You'll need to create a Slack app and generate a bot token:

1. Go to [Slack API](https://api.slack.com/apps)
2. Create a new app
3. Add the following bot token scopes:
   - `chat:write`
   - `chat:write.public`
4. Install the app to your workspace
5. Copy the Bot User OAuth Token (starts with `xoxb-`)

## Usage

Here's a basic example of how to use the extension in your k6 script:

```javascript
import slack from 'k6/x/slack';
import http from 'k6/http';
import { check } from 'k6';

// Configure Slack connection
slack.configure("your-slack-token", "your-channel-id");

export const options = {
    vus: 1,
    duration: '10s',
};

export function setup() {
    // Send a notification when the test starts
    slack.sendMessage("🧪 Performance test starting!");
}

export default function() {
    const res = http.get('https://test-api.example.com');
}

export function handleSummary(data) {
    // Format your test results
    const results = {
        status: "passed",
        testName: "API Performance Test",
        environment: __ENV.TEST_ENV || "staging",
        metrics: {
            "Response Time (avg)": `${data.metrics.http_req_duration.values.avg}ms`,
            "Success Rate": `${100 - (data.metrics.http_req_failed.values.rate * 100)}%`,
            // Add more metrics as needed
        }
    };

    // Send results to Slack
    slack.sendTestResults(JSON.stringify(results));
    return {};
}
```

## Available Functions

### `configure(token: string, channel: string)`
Configures the Slack client with your bot token and target channel.

### `sendMessage(message: string)`
Sends a simple text message to the configured Slack channel.

### `sendTestResults(resultsJSON: string)`
Sends formatted test results using Slack blocks. The results should be a JSON string with the following structure:

```javascript
{
    status: string,        // "passed" or "failed"
    testName: string,      // name of your test
    environment: string,   // environment being tested
    metrics: {            // object containing your metrics
        [key: string]: string | number
    }
}
```

## Environment Variables

- `TEST_ENV`: Specify the environment being tested (defaults to "staging" if not set)

## Development

To contribute to this project:

1. Create your feature branch
2. Run tests: `go test ./...`
3. Commit your changes
4. Push to your branch
5. Create a Pull Request

## License

MIT License

## Support

For issues and feature requests, please open an issue in the GitHub repository.. Build locally: `make build`
