package jira

import "encoding/json"

// WebhookPayload is the Jira issue webhook payload shape.
type WebhookPayload struct {
	Timestamp    int64  `json:"timestamp"`
	WebhookEvent string `json:"webhookEvent"`
	Issue        struct {
		ID     string                     `json:"id"`
		Key    string                     `json:"key"`
		Fields map[string]json.RawMessage `json:"fields"`
	} `json:"issue"`
}

// ExtractDocID reads the Google Doc ID from the payload using the configured field key.
// fieldKey is typically something like "customfield_10100" — the exact key depends on the Jira config.
func ExtractDocID(payload *WebhookPayload, fieldKey string) string {
	raw, ok := payload.Issue.Fields[fieldKey]
	if !ok {
		return ""
	}
	var docID string
	if err := json.Unmarshal(raw, &docID); err != nil {
		return ""
	}
	return docID
}
