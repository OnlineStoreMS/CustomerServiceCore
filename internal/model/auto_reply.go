package model

import "time"

type CsAutoReplyRule struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	TenantID    uint64    `gorm:"not null;index" json:"tenantId"`
	ShopID      uint64    `gorm:"index;not null;default:0" json:"shopId"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Enabled     bool      `gorm:"not null;default:true" json:"enabled"`
	MatchMode   string    `gorm:"size:16;not null;default:exact" json:"matchMode"`
	Keywords    string    `gorm:"type:text;not null" json:"keywords"`
	ReplyText   string    `gorm:"type:text;not null" json:"replyText"`
	Priority    int       `gorm:"not null;default:100" json:"priority"`
	CooldownSec int       `gorm:"not null;default:45" json:"cooldownSec"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (CsAutoReplyRule) TableName() string { return "cs_auto_reply_rules" }

type CsOutboundMessage struct {
	ID                   uint64     `gorm:"primaryKey" json:"id"`
	TenantID             uint64     `gorm:"not null;index" json:"tenantId"`
	ShopID               uint64     `gorm:"index;not null" json:"shopId"`
	ConversationID       uint64     `gorm:"index;not null" json:"conversationId"`
	Platform             string     `gorm:"size:32;not null" json:"platform"`
	PlatformShopID       string     `gorm:"size:64;not null" json:"platformShopId"`
	PlatformBuyerID      string     `gorm:"size:64;not null;index" json:"platformBuyerId"`
	BuyerName            string     `gorm:"size:128" json:"buyerName"`
	Content              string     `gorm:"type:text;not null" json:"content"`
	Source               string     `gorm:"size:32;not null;default:manual" json:"source"`
	RuleID               uint64     `json:"ruleId"`
	TriggerPlatformMsgID string     `gorm:"size:128;index" json:"triggerPlatformMessageId"`
	Status               string     `gorm:"size:16;not null;default:pending;index" json:"status"`
	Error                string     `gorm:"type:text" json:"error"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
	SentAt               *time.Time `json:"sentAt"`
}

func (CsOutboundMessage) TableName() string { return "cs_outbound_messages" }

type CsLlmSetting struct {
	ID                uint64    `gorm:"primaryKey" json:"id"`
	TenantID          uint64    `gorm:"not null;uniqueIndex" json:"tenantId"`
	Enabled           bool      `gorm:"not null;default:false" json:"enabled"`
	StyleHint         string    `gorm:"size:256" json:"styleHint"`
	CooldownSec       int       `gorm:"not null;default:25" json:"cooldownSec"`
	SystemPrompt      string    `gorm:"type:text" json:"systemPrompt"`
	Model             string    `gorm:"size:64" json:"model"`
	MaxChars          int       `gorm:"not null;default:80" json:"maxChars"`
	MaxTokens         int       `gorm:"not null;default:220" json:"maxTokens"`
	TimeoutSec        int       `gorm:"not null;default:12" json:"timeoutSec"`
	Temperature       float64   `gorm:"not null;default:0.35" json:"temperature"`
	ThinkingEnabled   bool      `gorm:"not null;default:false" json:"thinkingEnabled"`
	HistoryCount      int       `gorm:"not null;default:10" json:"historyCount"`
	InboundMaxChars   int       `gorm:"not null;default:80" json:"inboundMaxChars"`
	UseProductContext bool      `gorm:"not null;default:true" json:"useProductContext"`
	RetryStall        bool      `gorm:"not null;default:true" json:"retryStall"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func (CsLlmSetting) TableName() string { return "cs_llm_settings" }
