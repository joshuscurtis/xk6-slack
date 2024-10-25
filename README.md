# xk6-slack

This is a [k6](https://go.k6.io/k6) extension for sending test results to Slack using the xk6 system.

## Build

To build a `k6` binary with this extension, first ensure you have the prerequisites:

- [Go](https://go.dev/) 1.19+
- Git

Then:

1. Install `xk6`:
  ```bash
  go install go.k6.io/xk6/cmd/xk6@latest
  ```

2. Build the binary:
  ```bash
  xk6 build --with github.com/joshuscurtis/xk6-slack@latest
  ```

## Usage

```javascript
import slack from 'k6/x/slack';

const slackConfig = {
    slackChannelID: 'YOUR_CHANNEL_ID',
    dashboardUrls: {
        'K6 Dashboard': '/path-to/k6-dashboard',
    },
};

export function setup() {
    const slackClient = new slack.Client();
    slackClient.configure(
        __ENV.SLACK_TOKEN,
        slackConfig,
        __ENV.GITLAB_USER_LOGIN
    );
    slackClient.sendMessage('Start');
}
```

## Development

1. Clone the repo
2. Install dependencies: `make setup`
3. Run tests: `make test`
4. Build locally: `make build`

## License

Apache License 2.0
