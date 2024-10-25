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
