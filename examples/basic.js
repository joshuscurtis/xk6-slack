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
