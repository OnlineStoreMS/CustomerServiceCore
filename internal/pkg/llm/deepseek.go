package llm

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

const defaultSystemPrompt = `你是抖店飞鸽里正在打字的真人客服，不是助手，也不是AI。
只输出要发给买家的正文，不要解释。
规矩：
1. 一句，最多两句，总共不超过40个字
2. 口语，像微信聊天，不要书面语
3. 禁止分点、Markdown、括号补充、表情堆砌
4. 禁止这些说法：亲、亲爱的、您好、很高兴为您服务、希望对您有帮助、根据您的描述、如有其他问题随时
5. 不要编造单号、库存、价格、到货时间；不清楚就说「我帮你看下」或反问最短一句
6. 寒暄就短回，别展开`

type Client struct {
	enabled    bool
	apiBase    string
	apiKey     string
	model      string
	timeout    time.Duration
	maxChars   int
	httpClient *http.Client
}

func New(enabled bool, apiBase, apiKey, model string, timeoutSec, maxChars int) *Client {
	if apiBase == "" {
		apiBase = "https://api.deepseek.com"
	}
	if model == "" {
		model = "deepseek-chat"
	}
	if timeoutSec <= 0 {
		timeoutSec = 8
	}
	if maxChars <= 0 {
		maxChars = 40
	}
	return &Client{
		enabled:  enabled,
		apiBase:  strings.TrimRight(apiBase, "/"),
		apiKey:   strings.TrimSpace(apiKey),
		model:    model,
		timeout:  time.Duration(timeoutSec) * time.Second,
		maxChars: maxChars,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.enabled && c.apiKey != ""
}

func (c *Client) Model() string {
	if c == nil {
		return ""
	}
	return c.model
}

func (c *Client) MaxChars() int {
	if c == nil || c.maxChars <= 0 {
		return 40
	}
	return c.maxChars
}

type chatReq struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) Reply(ctx context.Context, shopName, styleHint, transcript string) (string, error) {
	if !c.Configured() {
		return "", fmt.Errorf("llm not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	sys := defaultSystemPrompt
	if hint := strings.TrimSpace(styleHint); hint != "" {
		sys += "\n额外口吻：" + hint
	}
	user := "店铺：" + strings.TrimSpace(shopName) + "\n最近对话：\n" + strings.TrimSpace(transcript) + "\n只回买家最后一句。"

	body, err := json.Marshal(chatReq{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: sys},
			{Role: "user", Content: user},
		},
		Temperature: 0.35,
		MaxTokens:   80,
	})
	if err != nil {
		return "", err
	}

	url := c.apiBase + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("deepseek %d: %s", res.StatusCode, strings.TrimSpace(string(raw)))
	}
	var parsed chatResp
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("deepseek: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("deepseek empty choices")
	}
	text := HumanizeReply(parsed.Choices[0].Message.Content, c.maxChars)
	if text == "" {
		return "", fmt.Errorf("reply sanitized empty")
	}
	return text, nil
}
