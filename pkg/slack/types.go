package slack

type MessageType string

const (
    StartMessage MessageType = "Start"
    EndMessage   MessageType = "End"
)

type DashboardURLs map[string]string
type GraphURLs map[string]string

type Config struct {
    SlackChannelID string        `json:"slackChannelID"`
    DashboardURLs  DashboardURLs `json:"dashboardUrls"`
    GraphURLs      GraphURLs     `json:"graphUrls"`
}
