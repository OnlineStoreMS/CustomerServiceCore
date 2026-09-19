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

const DefaultSystemPrompt = `你现在是一名耐心、专业的电商客服，正在抖店飞鸽里和买家打字。
不是AI助手，只输出要发给买家的正文，不要解释。
规矩：
1. 先看「买家正在看的商品」，后面的问题都按这个范围答，不要跳到别的品类
2. 结合整段对话回答，追问（什么区别、呢、这个、那款）要接上前面提到的型号或商品
3. 规格、型号对比、适不适合这类常识题，直接给结论，一两句说完，不要「帮你看下 / 稍等 / 对比下」
4. 口语，像微信聊天；禁止分点、Markdown、括号补充、表情堆砌
5. 禁止：亲、亲爱的、您好、很高兴为您服务、希望对您有帮助、根据您的描述、如有其他问题随时
6. 本店单号、库存、券后价、快递、到货时间不要编；只有这类才说「我帮你看下」
7. 客服已经说过稍等或帮你看下时，这轮必须给具体答案，禁止再拖
8. 寒暄就短回，别展开
9. 客服刚刚说过的意思不要换皮再发；买家没提出新问题就不要回`

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
		model = "deepseek-flash"
	}
	if timeoutSec <= 0 {
		timeoutSec = 20
	}
	if maxChars <= 0 {
		maxChars = 80
	}
	return &Client{
		enabled:  enabled,
		apiBase:  strings.TrimRight(apiBase, "/"),
		apiKey:   strings.TrimSpace(apiKey),
		model:    model,
		timeout:  time.Duration(timeoutSec) * time.Second,
		maxChars: maxChars,
		httpClient: &http.Client{
			// 单次请求由 Reply 的 context 超时控制；这里只挡死挂。
			Timeout: 60 * time.Second,
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
		return 80
	}
	return c.maxChars
}

type chatReq struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
	Thinking    *thinkingOpt  `json:"thinking,omitempty"`
}

type thinkingOpt struct {
	Type string `json:"type"`
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

type ReplyOptions struct {
	ShopName          string
	StyleHint         string
	ProductBackground string
	Transcript        string
	SystemPrompt      string
	Model             string
	MaxChars          int
	MaxTokens         int
	TimeoutSec        int
	Temperature       float64
	Thinking          bool
	UseProductContext bool
	RetryStall        bool
}

func (c *Client) Reply(ctx context.Context, opt ReplyOptions) (string, error) {
	if !c.Configured() {
		return "", fmt.Errorf("llm not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := c.timeout
	if opt.TimeoutSec > 0 {
		timeout = time.Duration(opt.TimeoutSec) * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sys := strings.TrimSpace(opt.SystemPrompt)
	if sys == "" {
		sys = DefaultSystemPrompt
	}
	if hint := strings.TrimSpace(opt.StyleHint); hint != "" {
		sys += "\n额外口吻：" + hint
	}
	needAnswer := opt.RetryStall && (stallAlreadySaid(opt.Transcript) || looksLikeProductQuestion(opt.Transcript))
	user := "店铺：" + strings.TrimSpace(opt.ShopName) + "\n"
	if opt.UseProductContext {
		if bg := strings.TrimSpace(opt.ProductBackground); bg != "" {
			user += "买家正在看的商品（飞鸽右侧咨询宝贝/浏览足迹，后面的问题都按这个范围答）：\n" + bg + "\n"
			if opt.RetryStall && (looksLikeProductQuestion(opt.Transcript) || stallAlreadySaid(opt.Transcript)) {
				needAnswer = true
			}
		}
	}
	user += "最近对话：\n" + strings.TrimSpace(opt.Transcript) + "\n结合整段对话直接回买家。"
	if needAnswer {
		user += "\n这轮必须给出具体区别或结论，禁止稍等、帮你看下、对比下这类空话。"
	}

	text, err := c.complete(ctx, sys, user, opt)
	if err != nil {
		return "", err
	}
	if needAnswer && LooksLikeStall(text) {
		if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < 8*time.Second {
			// 剩余时间不够再打一轮，按用户要求不强求，把第一句发出去。
			return text, nil
		}
		retry, retryErr := c.complete(ctx, sys, user+"\n上一句是空话，重写：用对话里的型号直接对比，给能发飞鸽的一句答案。", opt)
		if retryErr == nil && retry != "" && !LooksLikeStall(retry) {
			text = retry
		} else {
			return "", fmt.Errorf("stall reply")
		}
	}
	if text == "" {
		return "", fmt.Errorf("reply sanitized empty")
	}
	return text, nil
}

func (c *Client) complete(ctx context.Context, sys, user string, opt ReplyOptions) (string, error) {
	model := strings.TrimSpace(opt.Model)
	if model == "" {
		model = c.model
	}
	maxChars := opt.MaxChars
	if maxChars <= 0 {
		maxChars = c.maxChars
	}
	temp := opt.Temperature
	if temp <= 0 {
		temp = 0.35
	}
	maxTokens := opt.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 220
	}
	think := "disabled"
	if opt.Thinking {
		think = "enabled"
	}
	body, err := json.Marshal(chatReq{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: sys},
			{Role: "user", Content: user},
		},
		Temperature: temp,
		MaxTokens:   maxTokens,
		Thinking:    &thinkingOpt{Type: think},
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
	text := HumanizeReply(parsed.Choices[0].Message.Content, maxChars)
	if text == "" {
		return "", fmt.Errorf("reply sanitized empty")
	}
	return text, nil
}
