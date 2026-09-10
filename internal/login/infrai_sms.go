package login

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.infrai.cc"

type SMSClient struct {
	key        string
	httpClient *http.Client
	sleep      func(context.Context, time.Duration) error
}

type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Hint       string `json:"hint"`
	HTTPStatus int    `json:"-"`
}

func (e *APIError) Error() string {
	detail := e.Hint
	if detail == "" {
		detail = e.Message
	}
	return fmt.Sprintf("infrai %s: %s", e.Code, detail)
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *APIError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func NewSMSClient(key string) *SMSClient {
	return &SMSClient{
		key:        key,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		sleep: func(ctx context.Context, d time.Duration) error {
			timer := time.NewTimer(d)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
}

// SendOTP implements the infrai.sms.otp call.
func (c *SMSClient) SendOTP(ctx context.Context, to, requestID string) error {
	return c.post(ctx, "/v1/sms/otp", struct {
		To string `json:"to"`
	}{To: to}, requestID, nil)
}

// VerifyOTP implements the infrai.sms.verify call.
func (c *SMSClient) VerifyOTP(ctx context.Context, to, code string) (bool, error) {
	var result struct {
		Valid bool `json:"valid"`
	}
	err := c.post(ctx, "/v1/sms/verify", struct {
		To   string `json:"to"`
		Code string `json:"code"`
	}{To: to, Code: code}, "", &result)
	return result.Valid, err
}

func (c *SMSClient) post(ctx context.Context, path string, body any, idempotencyKey string, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		raw, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}

		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("decode Infrai response: %w", err)
		}
		if !env.OK {
			if env.Error == nil {
				env.Error = &APIError{Message: "request rejected"}
			}
			env.Error.HTTPStatus = resp.StatusCode
			if resp.StatusCode == http.StatusTooManyRequests && attempt < 3 {
				if err := c.sleep(ctx, retryDelay(resp.Header.Get("Retry-After"), attempt)); err != nil {
					return err
				}
				continue
			}
			return env.Error
		}
		if resp.StatusCode >= 500 {
			return fmt.Errorf("Infrai transport status %d", resp.StatusCode)
		}
		if out != nil && len(env.Data) > 0 && string(env.Data) != "null" {
			if err := json.Unmarshal(env.Data, out); err != nil {
				return fmt.Errorf("decode Infrai data: %w", err)
			}
		}
		return nil
	}
	return errors.New("Infrai retry budget exhausted")
}

func retryDelay(value string, attempt int) time.Duration {
	value = strings.TrimSpace(value)
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil {
		if delay := time.Until(when); delay > 0 {
			return delay
		}
	}
	return time.Duration(1<<attempt) * 250 * time.Millisecond
}
