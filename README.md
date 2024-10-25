# xk6-slack

xk6 extension for sending k6 test results to Slack.

## Build

1. Install xk6:
```bash
go install go.k6.io/xk6/cmd/xk6@latest
```

2. Build k6 with the extension:
```bash
xk6 build --with github.com/joshuscurtis/xk6-slack@latest
```

## Usage

```javascript
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
```

## Environment Variables

- SLACK_TOKEN: Slack Bot User OAuth Token
- SLACK_CHANNEL: Channel ID to post messages
- GITLAB_USER_LOGIN: Username to attribute the test run

## Development

1. Clone the repo
2. Install dependencies: `make setup`
3. Run tests: `make test`
4. Build locally: `make build`
