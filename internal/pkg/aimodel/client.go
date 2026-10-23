package aimodel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func NewClient(baseURL, token string, timeoutSec int) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	token = strings.TrimSpace(token)
	if baseURL == "" || token == "" {
		return nil
	}
	if timeoutSec <= 0 {
		timeoutSec = 90
	}
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTP:    &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

type VisionInput struct {
	TenantID uint64
	System   string
	Prompt   string
	ImageURL string
}

func (c *Client) Vision(ctx context.Context, in VisionInput) (content, model string, err error) {
	if c == nil {
		return "", "", fmt.Errorf("未配置 AI 大模型中心")
	}
	sys, err := json.Marshal(in.System)
	if err != nil {
		return "", "", err
	}
	user, err := json.Marshal([]any{
		map[string]string{"type": "text", "text": in.Prompt},
		map[string]any{"type": "image_url", "image_url": map[string]string{"url": in.ImageURL}},
	})
	if err != nil {
		return "", "", err
	}
	temp := 0.0
	thinking := false
	payload := map[string]any{
		"tenantId":     in.TenantID,
		"callerApp":    "customerservicecore",
		"capability":   "vision",
		"providerCode": "private",
		"temperature":  temp,
		"maxTokens":    2500,
		"thinking":     thinking,
		"messages": []map[string]any{
			{"role": "system", "content": json.RawMessage(sys)},
			{"role": "user", "content": json.RawMessage(user)},
		},
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v1/internal/chat", bytes.NewReader(buf))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", c.Token)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("调用私有识图模型失败")
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Content string `json:"content"`
			Model   string `json:"model"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return "", "", fmt.Errorf("识图模型返回异常")
	}
	if resp.StatusCode >= 300 || body.Code >= 300 {
		msg := strings.TrimSpace(body.Message)
		if msg == "" {
			msg = "识图失败"
		}
		return "", "", fmt.Errorf("%s", msg)
	}
	text := strings.TrimSpace(body.Data.Content)
	if text == "" {
		return "", "", fmt.Errorf("识图模型没有返回内容")
	}
	return text, strings.TrimSpace(body.Data.Model), nil
}
