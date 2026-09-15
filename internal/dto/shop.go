package dto

type ShopCreateInput struct {
	Name             string `json:"name" binding:"required"`
	Platform         string `json:"platform"`
	PlatformShopID   string `json:"platformShopId"`
	PlatformShopName string `json:"platformShopName"`
	Remark           string `json:"remark"`
}

type ShopUpdateInput struct {
	Name           *string `json:"name"`
	Remark         *string `json:"remark"`
	MonitorEnabled *bool   `json:"monitorEnabled"`
}

type ShopItem struct {
	ID               uint64 `json:"id"`
	Name             string `json:"name"`
	Platform         string `json:"platform"`
	PlatformLabel    string `json:"platformLabel"`
	PlatformShopID   string `json:"platformShopId,omitempty"`
	PlatformShopName string `json:"platformShopName,omitempty"`
	BindCode         string `json:"bindCode"`
	PluginStatus     string `json:"pluginStatus"`
	MonitorEnabled   bool   `json:"monitorEnabled"`
	LastSeenAt       string `json:"lastSeenAt,omitempty"`
	Remark           string `json:"remark,omitempty"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type PluginBindInput struct {
	BindCode string `json:"bindCode" binding:"required"`
}

type PluginBindResult struct {
	ShopID           uint64 `json:"shopId"`
	ShopName         string `json:"shopName"`
	Platform         string `json:"platform"`
	PlatformShopID   string `json:"platformShopId"`
	PlatformShopName string `json:"platformShopName"`
	PluginKey        string `json:"pluginKey"`
	PluginSecret     string `json:"pluginSecret"`
}

type PluginHeartbeatResult struct {
	MonitorEnabled bool   `json:"monitorEnabled"`
	ShopName       string `json:"shopName"`
	Platform       string `json:"platform"`
	PlatformShopID string `json:"platformShopId"`
}

type PluginMessagesInput struct {
	Platform         string              `json:"platform"`
	PlatformShopID   string              `json:"platformShopId"`
	PlatformShopName string              `json:"platformShopName"`
	CurrentListEmpty bool                `json:"currentListEmpty"`
	CurrentBuyers    []string            `json:"currentBuyers"`
	Messages         []PluginMessageItem `json:"messages"`
}

type PluginMessageItem struct {
	PlatformMessageID      string `json:"platformMessageId"`
	PlatformBuyerID        string `json:"platformBuyerId"`
	BuyerName              string `json:"buyerName"`
	PlatformConversationID string `json:"platformConversationId"`
	Direction              string `json:"direction"`
	Content                string `json:"content"`
	SentAt                 any    `json:"sentAt"`
	RawJSON                string `json:"rawJson"`
}

type PluginMessagesResult struct {
	Accepted int `json:"accepted"`
	Skipped  int `json:"skipped"`
}

type ConversationItem struct {
	ID                     uint64   `json:"id"`
	ShopID                 uint64   `json:"shopId"`
	ShopName               string   `json:"shopName,omitempty"`
	Platform               string   `json:"platform"`
	PlatformShopID         string   `json:"platformShopId"`
	PlatformShopName       string   `json:"platformShopName,omitempty"`
	PlatformBuyerID        string   `json:"platformBuyerId"`
	BuyerName              string   `json:"buyerName"`
	PlatformConversationID string   `json:"platformConversationId,omitempty"`
	LastMessageAt          string   `json:"lastMessageAt,omitempty"`
	LastMessagePreview     string   `json:"lastMessagePreview,omitempty"`
	UnreadHint             int      `json:"unreadHint"`
	MergedIDs              []uint64 `json:"mergedIds,omitempty"`
	CreatedAt              string   `json:"createdAt"`
	UpdatedAt              string   `json:"updatedAt"`
}

type MessageItem struct {
	ID                uint64 `json:"id"`
	ConversationID    uint64 `json:"conversationId"`
	PlatformMessageID string `json:"platformMessageId"`
	PlatformBuyerID   string `json:"platformBuyerId"`
	Direction         string `json:"direction"`
	Content           string `json:"content"`
	SentAt            string `json:"sentAt"`
	CreatedAt         string `json:"createdAt"`
}
