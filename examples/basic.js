import slack from 'k6/x/slack';

export const options = {
    vus: 1,
    duration: '30s',
};

// Initialize the slack module
slack.configure(ENV.SLACK_TOKEN, ENV.SLACK_CHANNEL_ID);

export function setup() {
    slack.sendMessage("🧪 Test starting!");
    return {};
}

export default function() {
    const res = http.get('https://www.google.com');

    check(res, {
      'is status 200': (r) => r.status === 200,
    });
}

export function handleSummary(data) {

    console.log(JSON.stringify(data, null, 2));
    slack.sendTestResults(data);
    return {};
}
