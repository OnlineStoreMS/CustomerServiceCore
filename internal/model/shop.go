package model

import "time"

const (
	PlatformDoudian = "doudian"

	PluginUnbound = "unbound"
	PluginBound   = "bound"
	PluginOnline  = "online"
	PluginOffline = "offline"

	DirectionIn  = "in"
	DirectionOut = "out"

	OutboundPending = "pending"
	OutboundSending = "sending"
	OutboundSent    = "sent"
	OutboundFailed  = "failed"

	ReplySourceManual = "manual"
	ReplySourceAuto   = "auto_reply"

	MatchExact    = "exact"
	MatchContains = "contains"
)

type CsShop struct {
	ID               uint64     `gorm:"primaryKey" json:"id"`
	TenantID         uint64     `gorm:"not null;uniqueIndex:uk_cs_shop_platform" json:"tenantId"`
	Name             string     `gorm:"size:128;not null" json:"name"`
	Platform         string     `gorm:"size:32;not null;default:doudian;uniqueIndex:uk_cs_shop_platform" json:"platform"`
	PlatformShopID   string     `gorm:"size:64;uniqueIndex:uk_cs_shop_platform" json:"platformShopId"`
	PlatformShopName string     `gorm:"size:128" json:"platformShopName"`
	BindCode         string     `gorm:"size:16;uniqueIndex;not null" json:"bindCode"`
	PluginKey        string     `gorm:"size:64;index" json:"pluginKey"`
	PluginSecretHash string     `gorm:"size:128" json:"-"`
	PluginSecretEnc  string     `gorm:"size:512" json:"-"`
	PluginStatus     string     `gorm:"size:32;not null;default:unbound" json:"pluginStatus"`
	MonitorEnabled   bool       `gorm:"not null;default:false" json:"monitorEnabled"`
	LastSeenAt       *time.Time `json:"lastSeenAt"`
	Remark           string     `gorm:"type:text" json:"remark"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func (CsShop) TableName() string { return "cs_shops" }

type CsConversation struct {
	ID                     uint64     `gorm:"primaryKey" json:"id"`
	TenantID               uint64     `gorm:"not null;uniqueIndex:uk_cs_conv_buyer" json:"tenantId"`
	ShopID                 uint64     `gorm:"index;not null" json:"shopId"`
	Platform               string     `gorm:"size:32;not null;uniqueIndex:uk_cs_conv_buyer" json:"platform"`
	PlatformShopID         string     `gorm:"size:64;uniqueIndex:uk_cs_conv_buyer" json:"platformShopId"`
	PlatformBuyerID        string     `gorm:"size:64;uniqueIndex:uk_cs_conv_buyer" json:"platformBuyerId"`
	BuyerName              string     `gorm:"size:128" json:"buyerName"`
	PlatformConversationID string     `gorm:"size:128" json:"platformConversationId"`
	LastMessageAt          *time.Time `json:"lastMessageAt"`
	LastMessagePreview     string     `gorm:"size:512" json:"lastMessagePreview"`
	UnreadHint             int        `gorm:"not null;default:0" json:"unreadHint"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
}

func (CsConversation) TableName() string { return "cs_conversations" }

type CsMessage struct {
	ID                uint64    `gorm:"primaryKey" json:"id"`
	TenantID          uint64    `gorm:"not null;uniqueIndex:uk_cs_msg_platform" json:"tenantId"`
	ShopID            uint64    `gorm:"index;not null" json:"shopId"`
	ConversationID    uint64    `gorm:"index;not null" json:"conversationId"`
	Platform          string    `gorm:"size:32;not null;uniqueIndex:uk_cs_msg_platform" json:"platform"`
	PlatformMessageID string    `gorm:"size:128;uniqueIndex:uk_cs_msg_platform" json:"platformMessageId"`
	PlatformShopID    string    `gorm:"size:64" json:"platformShopId"`
	PlatformBuyerID   string    `gorm:"size:64;index" json:"platformBuyerId"`
	Direction         string    `gorm:"size:8;not null" json:"direction"`
	Content           string    `gorm:"type:text" json:"content"`
	SentAt            time.Time `gorm:"index" json:"sentAt"`
	RawJSON           string    `gorm:"type:text" json:"rawJson"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func (CsMessage) TableName() string { return "cs_messages" }
