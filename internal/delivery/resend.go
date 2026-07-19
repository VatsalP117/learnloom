package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	APIKey        string
	From          string
	SubjectPrefix string
	Endpoint      string
	Timeout       time.Duration
}

type Resend struct {
	cfg    Config
	client *http.Client
}

type Message struct {
	IssueID      string
	GenerationID string
	To           string
	Subject      string
	HTML         string
	Text         string
}

type OutcomeUnknownError struct {
	Cause error
}

func (e *OutcomeUnknownError) Error() string {
	return "delivery outcome is unknown"
}

func (e *OutcomeUnknownError) Unwrap() error {
	return e.Cause
}

func NewResend(cfg Config) (*Resend, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("Resend API key is required")
	}
	if strings.TrimSpace(cfg.From) == "" {
		return nil, errors.New("Resend sender is required")
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://api.resend.com/emails"
	}
	if cfg.SubjectPrefix == "" {
		cfg.SubjectPrefix = "Learnloom"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Resend{cfg: cfg, client: &http.Client{Timeout: cfg.Timeout}}, nil
}

func (r *Resend) Deliver(ctx context.Context, message Message) (string, error) {
	if strings.TrimSpace(message.IssueID) == "" ||
		strings.TrimSpace(message.GenerationID) == "" {
		return "", errors.New("delivery identity is required")
	}
	if strings.TrimSpace(message.To) == "" || !strings.Contains(message.To, "@") {
		return "", errors.New("delivery recipient is invalid")
	}
	subject := strings.TrimSpace(message.Subject)
	if subject == "" {
		subject = "Daily Dossier"
	}
	payload := struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		HTML    string   `json:"html"`
		Text    string   `json:"text"`
	}{
		From: r.cfg.From, To: []string{message.To},
		Subject: safeSubject(r.cfg.SubjectPrefix + " — " + subject),
		HTML:    message.HTML, Text: message.Text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		r.cfg.Endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+r.cfg.APIKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(
		"Idempotency-Key",
		"learnloom/"+message.IssueID+"/"+message.GenerationID,
	)
	response, err := r.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", &OutcomeUnknownError{Cause: err}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 8<<10))
		return "", fmt.Errorf(
			"Resend returned HTTP %d%s",
			response.StatusCode,
			safeDetail(string(detail), r.cfg.APIKey),
		)
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return "", &OutcomeUnknownError{Cause: fmt.Errorf("decode Resend response: %w", err)}
	}
	if strings.TrimSpace(result.ID) == "" {
		return "", &OutcomeUnknownError{Cause: errors.New("Resend returned no email identifier")}
	}
	return result.ID, nil
}

func safeSubject(value string) string {
	value = strings.Join(strings.Fields(strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " ")), " ")
	runes := []rune(value)
	if len(runes) > 180 {
		value = string(runes[:180])
	}
	return value
}

func safeDetail(value, secret string) string {
	value = strings.ReplaceAll(value, secret, "[REDACTED]")
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > 800 {
		value = string(runes[:800])
	}
	if value == "" {
		return ""
	}
	return ": " + value
}
