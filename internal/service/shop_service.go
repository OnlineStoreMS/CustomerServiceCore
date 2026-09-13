package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"customerservicecore/internal/dto"
	"customerservicecore/internal/model"
	"customerservicecore/internal/pkg/pluginsecret"
	"customerservicecore/internal/repo"

	"gorm.io/gorm"
)

var platformLabels = map[string]string{
	model.PlatformDoudian: "抖店",
}

type ShopService struct {
	repos  *repo.Repos
	tenant uint64
	codec  *pluginsecret.Codec
}

func NewShopService(repos *repo.Repos, codec *pluginsecret.Codec) *ShopService {
	return &ShopService{repos: repos, codec: codec}
}

func (s *ShopService) ForTenant(tenantID uint64) *ShopService {
	return &ShopService{
		repos:  s.repos,
		tenant: repo.NormalizeTenantID(tenantID),
		codec:  s.codec,
	}
}

func (s *ShopService) shops() *repo.ShopRepo {
	return s.repos.Shop.ForTenant(s.tenant)
}

func (s *ShopService) conversations() *repo.ConversationRepo {
	return s.repos.Conversation.ForTenant(s.tenant)
}

func (s *ShopService) messages() *repo.MessageRepo {
	return s.repos.Message.ForTenant(s.tenant)
}

func (s *ShopService) List() ([]dto.ShopItem, error) {
	list, err := s.shops().List()
	if err != nil {
		return nil, err
	}
	out := make([]dto.ShopItem, 0, len(list))
	for i := range list {
		out = append(out, s.toItem(&list[i]))
	}
	return out, nil
}

func (s *ShopService) Get(id uint64) (*dto.ShopItem, error) {
	shop, err := s.shops().Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	item := s.toItem(shop)
	return &item, nil
}

func (s *ShopService) Create(in *dto.ShopCreateInput) (*dto.ShopItem, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrBadRequest
	}
	platform := strings.TrimSpace(in.Platform)
	if platform == "" {
		platform = model.PlatformDoudian
	}
	if _, ok := platformLabels[platform]; !ok {
		return nil, fmt.Errorf("%w: 不支持的平台", ErrBadRequest)
	}
	platformShopID := strings.TrimSpace(in.PlatformShopID)
	if platformShopID == "" {
		return nil, fmt.Errorf("%w: 请填写平台店铺 ID", ErrBadRequest)
	}

	var shop *model.CsShop
	var lastErr error
	for i := 0; i < 6; i++ {
		code, err := randomBindCode()
		if err != nil {
			return nil, err
		}
		shop = &model.CsShop{
			Name:             name,
			Platform:         platform,
			BindCode:         code,
			PluginStatus:     model.PluginUnbound,
			PlatformShopID:   platformShopID,
			PlatformShopName: strings.TrimSpace(in.PlatformShopName),
			Remark:           strings.TrimSpace(in.Remark),
			MonitorEnabled:   false,
		}
		lastErr = s.shops().Create(shop)
		if lastErr == nil {
			item := s.toItem(shop)
			return &item, nil
		}
		lower := strings.ToLower(lastErr.Error())
		if !strings.Contains(lower, "unique") && !strings.Contains(lower, "duplicate") {
			return nil, lastErr
		}
	}
	return nil, lastErr
}

func (s *ShopService) Update(id uint64, in *dto.ShopUpdateInput) (*dto.ShopItem, error) {
	shop, err := s.shops().Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		if v := strings.TrimSpace(*in.Name); v != "" {
			shop.Name = v
		}
	}
	if in.Remark != nil {
		shop.Remark = strings.TrimSpace(*in.Remark)
	}
	if in.MonitorEnabled != nil {
		shop.MonitorEnabled = *in.MonitorEnabled
	}
	if err := s.shops().Save(shop); err != nil {
		return nil, err
	}
	item := s.toItem(shop)
	return &item, nil
}

func (s *ShopService) RotateBindCode(id uint64) (*dto.ShopItem, error) {
	shop, err := s.shops().Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	code, err := randomBindCode()
	if err != nil {
		return nil, err
	}
	shop.BindCode = code
	clearPluginCreds(shop)
	if err := s.shops().Save(shop); err != nil {
		return nil, err
	}
	item := s.toItem(shop)
	return &item, nil
}

func (s *ShopService) ResetPlugin(id uint64) (*dto.ShopItem, error) {
	shop, err := s.shops().Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	clearPluginCreds(shop)
	if err := s.shops().Save(shop); err != nil {
		return nil, err
	}
	item := s.toItem(shop)
	return &item, nil
}

func clearPluginCreds(shop *model.CsShop) {
	shop.PluginKey = ""
	shop.PluginSecretHash = ""
	shop.PluginSecretEnc = ""
	shop.PluginStatus = model.PluginUnbound
	shop.LastSeenAt = nil
}

func (s *ShopService) Bind(bindCode string) (*dto.PluginBindResult, error) {
	code := strings.ToUpper(strings.TrimSpace(bindCode))
	if len(code) < 4 {
		return nil, ErrBindCodeInvalid
	}
	shop, err := s.repos.Shop.GetByBindCode(code)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrBindCodeInvalid
	}
	if err != nil {
		return nil, err
	}
	if shop.PluginKey != "" {
		return nil, ErrAlreadyBound
	}
	if err := s.ForTenant(shop.TenantID).issuePluginCredentials(shop); err != nil {
		return nil, err
	}
	secret, err := s.codec.Decrypt(shop.PluginSecretEnc)
	if err != nil {
		return nil, err
	}
	return &dto.PluginBindResult{
		ShopID:           shop.ID,
		ShopName:         shop.Name,
		Platform:         shop.Platform,
		PlatformShopID:   shop.PlatformShopID,
		PlatformShopName: shop.PlatformShopName,
		PluginKey:        shop.PluginKey,
		PluginSecret:     secret,
	}, nil
}

func (s *ShopService) issuePluginCredentials(shop *model.CsShop) error {
	key, err := randomHex(16)
	if err != nil {
		return err
	}
	secret, err := randomHex(24)
	if err != nil {
		return err
	}
	if s.codec == nil {
		return fmt.Errorf("plugin secret codec 未初始化")
	}
	enc, err := s.codec.Encrypt(secret)
	if err != nil {
		return err
	}
	now := time.Now()
	shop.PluginKey = key
	shop.PluginSecretHash = hashSecret(secret)
	shop.PluginSecretEnc = enc
	shop.PluginStatus = model.PluginBound
	shop.LastSeenAt = &now
	return s.shops().Save(shop)
}

func (s *ShopService) AuthenticatePlugin(key, secret string) (*model.CsShop, error) {
	key = strings.TrimSpace(key)
	secret = strings.TrimSpace(secret)
	if key == "" || secret == "" {
		return nil, ErrPluginAuth
	}
	shop, err := s.repos.Shop.GetByPluginKey(key)
	if err != nil {
		return nil, ErrPluginAuth
	}
	got := hashSecret(secret)
	if subtle.ConstantTimeCompare([]byte(shop.PluginSecretHash), []byte(got)) != 1 {
		return nil, ErrPluginAuth
	}
	return shop, nil
}

func (s *ShopService) Heartbeat(shop *model.CsShop) (*dto.PluginHeartbeatResult, error) {
	now := time.Now()
	shop.LastSeenAt = &now
	shop.PluginStatus = model.PluginOnline
	if err := s.repos.Shop.TouchHeartbeat(shop); err != nil {
		return nil, err
	}
	return &dto.PluginHeartbeatResult{
		MonitorEnabled: shop.MonitorEnabled,
		ShopName:       shop.Name,
		Platform:       shop.Platform,
		PlatformShopID: shop.PlatformShopID,
	}, nil
}

func (s *ShopService) IngestMessages(shop *model.CsShop, in *dto.PluginMessagesInput) (*dto.PluginMessagesResult, error) {
	platform := strings.TrimSpace(in.Platform)
	if platform == "" {
		platform = shop.Platform
	}
	if platform == "" {
		platform = model.PlatformDoudian
	}
	platformShopID := strings.TrimSpace(in.PlatformShopID)
	if platformShopID == "" {
		platformShopID = shop.PlatformShopID
	}
	if platformShopID == "" {
		return nil, fmt.Errorf("%w: platformShopId 必填", ErrBadRequest)
	}
	if shop.PlatformShopID != "" && !strings.EqualFold(shop.PlatformShopID, platformShopID) {
		return nil, fmt.Errorf("%w: 上报店铺与绑定店铺不一致", ErrBadRequest)
	}
	if name := strings.TrimSpace(in.PlatformShopName); name != "" && shop.PlatformShopName == "" {
		shop.PlatformShopName = name
		_ = s.repos.Shop.ForTenant(shop.TenantID).Save(shop)
	}

	svc := s.ForTenant(shop.TenantID)
	accepted, skipped := 0, 0
	for _, item := range in.Messages {
		msgID := strings.TrimSpace(item.PlatformMessageID)
		buyerID := strings.TrimSpace(item.PlatformBuyerID)
		if msgID == "" || buyerID == "" {
			skipped++
			continue
		}
		dir := strings.TrimSpace(item.Direction)
		if dir != model.DirectionIn && dir != model.DirectionOut {
			skipped++
			continue
		}
		sentAt := parseSentAt(item.SentAt)
		content := strings.TrimSpace(item.Content)
		preview := truncateRunes(content, 200)

		conv, err := svc.ensureConversation(shop, platform, platformShopID, buyerID, item.BuyerName, item.PlatformConversationID, sentAt, preview)
		if err != nil {
			return nil, err
		}

		msg := &model.CsMessage{
			ShopID:            shop.ID,
			ConversationID:    conv.ID,
			Platform:          platform,
			PlatformMessageID: msgID,
			PlatformShopID:    platformShopID,
			PlatformBuyerID:   buyerID,
			Direction:         dir,
			Content:           content,
			SentAt:            sentAt,
			RawJSON:           item.RawJSON,
		}
		created, err := svc.messages().CreateIgnoreDuplicate(msg)
		if err != nil {
			return nil, err
		}
		if created {
			accepted++
			if conv.LastMessageAt == nil || sentAt.After(*conv.LastMessageAt) {
				conv.LastMessageAt = &sentAt
				conv.LastMessagePreview = preview
				if name := strings.TrimSpace(item.BuyerName); name != "" {
					conv.BuyerName = name
				}
				if cid := strings.TrimSpace(item.PlatformConversationID); cid != "" {
					conv.PlatformConversationID = cid
				}
				_ = svc.conversations().Save(conv)
			}
		} else {
			skipped++
		}
	}
	return &dto.PluginMessagesResult{Accepted: accepted, Skipped: skipped}, nil
}

func (s *ShopService) ensureConversation(
	shop *model.CsShop,
	platform, platformShopID, buyerID, buyerName, platformConvID string,
	sentAt time.Time,
	preview string,
) (*model.CsConversation, error) {
	conv, err := s.conversations().GetByBuyer(platform, platformShopID, buyerID)
	if err == nil {
		return conv, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	conv = &model.CsConversation{
		ShopID:                 shop.ID,
		Platform:               platform,
		PlatformShopID:         platformShopID,
		PlatformBuyerID:        buyerID,
		BuyerName:              strings.TrimSpace(buyerName),
		PlatformConversationID: strings.TrimSpace(platformConvID),
		LastMessageAt:          &sentAt,
		LastMessagePreview:     preview,
	}
	if err := s.conversations().UpsertByBuyer(conv); err != nil {
		return nil, err
	}
	// reload to get ID when conflict path ran
	if conv.ID == 0 {
		return s.conversations().GetByBuyer(platform, platformShopID, buyerID)
	}
	return conv, nil
}

func (s *ShopService) ListConversations(shopID uint64, page, pageSize int) ([]dto.ConversationItem, int64, error) {
	list, total, err := s.conversations().List(shopID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.ConversationItem, 0, len(list))
	for i := range list {
		out = append(out, toConversationItem(&list[i]))
	}
	return out, total, nil
}

func (s *ShopService) ListMessages(conversationID uint64, page, pageSize int) ([]dto.MessageItem, int64, error) {
	if _, err := s.conversations().Get(conversationID); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, 0, ErrNotFound
	} else if err != nil {
		return nil, 0, err
	}
	list, total, err := s.messages().ListByConversation(conversationID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.MessageItem, 0, len(list))
	for i := range list {
		out = append(out, toMessageItem(&list[i]))
	}
	return out, total, nil
}

func (s *ShopService) toItem(shop *model.CsShop) dto.ShopItem {
	item := dto.ShopItem{
		ID: shop.ID, Name: shop.Name, Platform: shop.Platform,
		PlatformLabel:    platformLabel(shop.Platform),
		BindCode:         shop.BindCode,
		PluginStatus:     shop.PluginStatus,
		MonitorEnabled:   shop.MonitorEnabled,
		PlatformShopID:   shop.PlatformShopID,
		PlatformShopName: shop.PlatformShopName,
		Remark:           shop.Remark,
		CreatedAt:        formatTime(shop.CreatedAt),
		UpdatedAt:        formatTime(shop.UpdatedAt),
	}
	if shop.LastSeenAt != nil {
		item.LastSeenAt = formatTime(*shop.LastSeenAt)
	}
	return item
}

func toConversationItem(c *model.CsConversation) dto.ConversationItem {
	item := dto.ConversationItem{
		ID: c.ID, ShopID: c.ShopID, Platform: c.Platform,
		PlatformShopID: c.PlatformShopID, PlatformBuyerID: c.PlatformBuyerID,
		BuyerName: c.BuyerName, PlatformConversationID: c.PlatformConversationID,
		LastMessagePreview: c.LastMessagePreview, UnreadHint: c.UnreadHint,
		CreatedAt: formatTime(c.CreatedAt), UpdatedAt: formatTime(c.UpdatedAt),
	}
	if c.LastMessageAt != nil {
		item.LastMessageAt = formatTime(*c.LastMessageAt)
	}
	return item
}

func toMessageItem(m *model.CsMessage) dto.MessageItem {
	return dto.MessageItem{
		ID: m.ID, ConversationID: m.ConversationID,
		PlatformMessageID: m.PlatformMessageID, PlatformBuyerID: m.PlatformBuyerID,
		Direction: m.Direction, Content: m.Content,
		SentAt: formatTime(m.SentAt), CreatedAt: formatTime(m.CreatedAt),
	}
}

func platformLabel(platform string) string {
	if v, ok := platformLabels[platform]; ok {
		return v
	}
	return platform
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

func parseSentAt(v any) time.Time {
	now := time.Now()
	if v == nil {
		return now
	}
	switch x := v.(type) {
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return now
		}
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return millisOrSeconds(n)
		}
	case float64:
		return millisOrSeconds(int64(x))
	case int64:
		return millisOrSeconds(x)
	case int:
		return millisOrSeconds(int64(x))
	}
	return now
}

func millisOrSeconds(n int64) time.Time {
	if n > 1_000_000_000_000 {
		return time.UnixMilli(n)
	}
	return time.Unix(n, 0)
}

func truncateRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}

func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// 8-char uppercase hex bind code
func randomBindCode() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(b)), nil
}
