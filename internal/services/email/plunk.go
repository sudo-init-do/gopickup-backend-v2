package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopickup/internal/config"
	"io"
	"net/http"
	"strings"
	"time"
)

type EmailService interface {
	SendEmail(to string, subject string, html string, text string) error
}

type PlunkService struct {
	apiKey    string
	fromEmail string
	fromName  string
	client    *http.Client
	apiURL    string
}

type plunkRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Name    string `json:"name,omitempty"`
	From    string `json:"from,omitempty"`
}

func NewPlunkService(cfg *config.Config) *PlunkService {
	return &PlunkService{
		apiKey:    strings.TrimSpace(cfg.PlunkAPIKey),
		fromEmail: cfg.PlunkFromEmail,
		fromName:  cfg.PlunkFromName,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		apiURL: "https://api.useplunk.com/v1/send",
	}
}

func (s *PlunkService) SendEmail(to string, subject string, html string, text string) error {
	// Plunk requires 'body' as HTML. If html is empty, wrap text in simple HTML.
	bodyContent := html
	if bodyContent == "" {
		// Basic escaping could be added here if needed, but for now simple wrapping
		bodyContent = fmt.Sprintf("<div>%s</div>", text)
	}

	reqBody := plunkRequest{
		To:      to,
		Subject: subject,
		Body:    bodyContent,
		Name:    s.fromName,
		From:    s.fromEmail,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", s.apiURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to Plunk: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("plunk API returned error status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
