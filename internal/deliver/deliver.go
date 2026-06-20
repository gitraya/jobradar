// Package deliver sends the rendered digest to Discord (webhook) and/or Gmail
// (SMTP). Secrets come from the environment, never the config file.
package deliver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"os"
	"strings"
)

// discordLimit is Discord's per-message content cap; we stay safely under 2000.
const discordLimit = 1900

// Discord posts the markdown digest to the webhook in DISCORD_WEBHOOK_URL,
// splitting it into messages that respect Discord's character limit.
func Discord(ctx context.Context, client *http.Client, markdown string) error {
	webhook := os.Getenv("DISCORD_WEBHOOK_URL")
	if webhook == "" {
		return fmt.Errorf("DISCORD_WEBHOOK_URL is not set")
	}
	for _, chunk := range chunkLines(markdown, discordLimit) {
		body, _ := json.Marshal(map[string]string{"content": chunk})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			return fmt.Errorf("discord webhook: status %d: %s", resp.StatusCode, respBody)
		}
	}
	return nil
}

// Gmail sends the HTML digest over Gmail SMTP. It reads GMAIL_USERNAME (the
// sending address), GMAIL_APP_PASSWORD (a Gmail App Password) and GMAIL_TO
// (comma-separated recipients; defaults to the sender).
func Gmail(htmlBody, subject string) error {
	user := os.Getenv("GMAIL_USERNAME")
	pass := os.Getenv("GMAIL_APP_PASSWORD")
	if user == "" || pass == "" {
		return fmt.Errorf("GMAIL_USERNAME and GMAIL_APP_PASSWORD must be set")
	}
	to := os.Getenv("GMAIL_TO")
	if to == "" {
		to = user
	}
	recipients := splitTrim(to)

	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s\r\n", user)
	fmt.Fprintf(&msg, "To: %s\r\n", strings.Join(recipients, ", "))
	fmt.Fprintf(&msg, "Subject: %s\r\n", subject)
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n")
	msg.WriteString(htmlBody)

	auth := smtp.PlainAuth("", user, pass, "smtp.gmail.com")
	return smtp.SendMail("smtp.gmail.com:587", auth, user, recipients, msg.Bytes())
}

// chunkLines splits text into pieces no longer than limit, breaking on line
// boundaries so markdown links are never cut mid-token.
func chunkLines(text string, limit int) []string {
	var chunks []string
	var cur strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if cur.Len()+len(line)+1 > limit && cur.Len() > 0 {
			chunks = append(chunks, cur.String())
			cur.Reset()
		}
		cur.WriteString(line)
		cur.WriteByte('\n')
	}
	if strings.TrimSpace(cur.String()) != "" {
		chunks = append(chunks, cur.String())
	}
	return chunks
}

func splitTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
