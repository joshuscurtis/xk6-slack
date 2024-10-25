import slack from 'k6/x/slack';
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
    vus: 10,
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
slackClient.configure(SLACK_TOKEN, slackConfig, USER);

export function setup() {
    slackClient.sendMessage('Start');
}

export default function () {
    const response = http.get('https://test.k6.io');
    check(response, {
        'status is 200': (r) => r.status === 200,
    });
    sleep(1);
}

export function handleSummary(data) {
    slackClient.sendMessage('End');
    return {};
}
