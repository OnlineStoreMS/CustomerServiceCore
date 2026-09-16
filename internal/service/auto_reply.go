package service

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"customerservicecore/internal/dto"
	"customerservicecore/internal/model"
	"customerservicecore/internal/repo"

	"gorm.io/gorm"
)

var replyPunctRe = regexp.MustCompile(`[~～!！?？.。,，、;；:：'"“”‘’()（）\[\]【】\s]+`)

func (s *ShopService) rules() *repo.AutoReplyRepo {
	return s.repos.AutoReply.ForTenant(s.tenant)
}

func (s *ShopService) outbound() *repo.OutboundRepo {
	return s.repos.Outbound.ForTenant(s.tenant)
}

func normalizeReplyText(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.ToLower(s)
	s = replyPunctRe.ReplaceAllString(s, "")
	return s
}

func splitKeywords(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；' || r == '\n' || r == '|'
	})
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		n := normalizeReplyText(p)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

func matchAutoReply(content, matchMode string, keywords []string) bool {
	got := normalizeReplyText(content)
	if got == "" || utf8.RuneCountInString(got) > 24 {
		return false
	}
	mode := strings.ToLower(strings.TrimSpace(matchMode))
	if mode == "" {
		mode = model.MatchExact
	}
	for _, kw := range keywords {
		if kw == "" {
			continue
		}
		switch mode {
		case model.MatchContains:
			if strings.Contains(got, kw) {
				return true
			}
		default:
			if got == kw {
				return true
			}
		}
	}
	return false
}

func (s *ShopService) ListAutoReplyRules() ([]dto.AutoReplyRuleItem, error) {
	list, err := s.rules().List()
	if err != nil {
		return nil, err
	}
	shopByID := map[uint64]string{}
	if shops, err := s.shops().List(); err == nil {
		for i := range shops {
			shopByID[shops[i].ID] = shops[i].Name
		}
	}
	out := make([]dto.AutoReplyRuleItem, 0, len(list))
	for i := range list {
		item := toAutoReplyItem(&list[i])
		if list[i].ShopID == 0 {
			item.ShopName = "全部店铺"
		} else if name, ok := shopByID[list[i].ShopID]; ok {
			item.ShopName = name
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *ShopService) CreateAutoReplyRule(in *dto.AutoReplyRuleInput) (*dto.AutoReplyRuleItem, error) {
	row, err := s.buildRule(in, nil)
	if err != nil {
		return nil, err
	}
	if err := s.rules().Create(row); err != nil {
		return nil, err
	}
	item := toAutoReplyItem(row)
	return &item, nil
}

func (s *ShopService) UpdateAutoReplyRule(id uint64, in *dto.AutoReplyRuleInput) (*dto.AutoReplyRuleItem, error) {
	row, err := s.rules().Get(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	next, err := s.buildRule(in, row)
	if err != nil {
		return nil, err
	}
	next.ID = row.ID
	next.TenantID = row.TenantID
	next.CreatedAt = row.CreatedAt
	if err := s.rules().Save(next); err != nil {
		return nil, err
	}
	item := toAutoReplyItem(next)
	return &item, nil
}

func (s *ShopService) DeleteAutoReplyRule(id uint64) error {
	if _, err := s.rules().Get(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return s.rules().Delete(id)
}

func (s *ShopService) SeedAutoReplyPresets() ([]dto.AutoReplyRuleItem, error) {
	n, err := s.rules().Count()
	if err != nil {
		return nil, err
	}
	if n > 0 {
		return s.ListAutoReplyRules()
	}
	presets := []model.CsAutoReplyRule{
		{
			Name:        "寒暄收到",
			Enabled:     true,
			MatchMode:   model.MatchExact,
			Keywords:    "好的,好的谢谢,好的呢,谢谢,谢谢老板,收到,嗯嗯,好,ok,OK,好哒,好的亲",
			ReplyText:   "好的亲，收到啦～有其他问题随时找我",
			Priority:    10,
			CooldownSec: 45,
		},
		{
			Name:        "告别",
			Enabled:     true,
			MatchMode:   model.MatchExact,
			Keywords:    "再见,拜拜,拜,没事了",
			ReplyText:   "好的，祝您生活愉快～",
			Priority:    20,
			CooldownSec: 45,
		},
	}
	for i := range presets {
		if err := s.rules().Create(&presets[i]); err != nil {
			return nil, err
		}
	}
	return s.ListAutoReplyRules()
}

func (s *ShopService) buildRule(in *dto.AutoReplyRuleInput, existing *model.CsAutoReplyRule) (*model.CsAutoReplyRule, error) {
	row := &model.CsAutoReplyRule{}
	if existing != nil {
		*row = *existing
	}
	if in.Name != "" || existing == nil {
		row.Name = strings.TrimSpace(in.Name)
	}
	if row.Name == "" {
		return nil, fmt.Errorf("%w: 请填写规则名称", ErrBadRequest)
	}
	if in.Keywords != "" || existing == nil {
		row.Keywords = strings.TrimSpace(in.Keywords)
	}
	if len(splitKeywords(row.Keywords)) == 0 {
		return nil, fmt.Errorf("%w: 请填写至少一个关键词", ErrBadRequest)
	}
	if in.ReplyText != "" || existing == nil {
		row.ReplyText = strings.TrimSpace(in.ReplyText)
	}
	if row.ReplyText == "" {
		return nil, fmt.Errorf("%w: 请填写回复内容", ErrBadRequest)
	}
	if in.MatchMode != "" || existing == nil {
		mode := strings.ToLower(strings.TrimSpace(in.MatchMode))
		if mode == "" {
			mode = model.MatchExact
		}
		if mode != model.MatchExact && mode != model.MatchContains {
			return nil, fmt.Errorf("%w: 匹配方式仅支持 exact / contains", ErrBadRequest)
		}
		row.MatchMode = mode
	}
	if in.ShopID != nil {
		row.ShopID = *in.ShopID
	}
	if in.Enabled != nil {
		row.Enabled = *in.Enabled
	} else if existing == nil {
		row.Enabled = true
	}
	if in.Priority != nil {
		row.Priority = *in.Priority
	} else if existing == nil {
		row.Priority = 100
	}
	if in.CooldownSec != nil {
		row.CooldownSec = *in.CooldownSec
	} else if existing == nil {
		row.CooldownSec = 45
	}
	if row.CooldownSec < 0 {
		row.CooldownSec = 0
	}
	return row, nil
}

func (s *ShopService) ReplyConversation(conversationID uint64, content string) (*dto.ConversationReplyResult, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("%w: 回复内容不能为空", ErrBadRequest)
	}
	conv, err := s.conversations().Get(conversationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	shop, err := s.shops().Get(conv.ShopID)
	if err != nil {
		return nil, err
	}
	out, msg, err := s.enqueueOutbound(shop, conv, content, model.ReplySourceManual, 0, "")
	if err != nil {
		return nil, err
	}
	return &dto.ConversationReplyResult{
		OutboundID: out.ID,
		Status:     out.Status,
		Message:    toMessageItem(msg),
	}, nil
}

func (s *ShopService) enqueueOutbound(
	shop *model.CsShop,
	conv *model.CsConversation,
	content, source string,
	ruleID uint64,
	triggerID string,
) (*model.CsOutboundMessage, *model.CsMessage, error) {
	now := time.Now()
	row := &model.CsOutboundMessage{
		ShopID:               shop.ID,
		ConversationID:       conv.ID,
		Platform:             conv.Platform,
		PlatformShopID:       conv.PlatformShopID,
		PlatformBuyerID:      conv.PlatformBuyerID,
		BuyerName:            conv.BuyerName,
		Content:              content,
		Source:               source,
		RuleID:               ruleID,
		TriggerPlatformMsgID: triggerID,
		Status:               model.OutboundPending,
	}
	if err := s.outbound().Create(row); err != nil {
		return nil, nil, err
	}
	msg := &model.CsMessage{
		ShopID:            shop.ID,
		ConversationID:    conv.ID,
		Platform:          conv.Platform,
		PlatformMessageID: fmt.Sprintf("cloud-out:%d", row.ID),
		PlatformShopID:    conv.PlatformShopID,
		PlatformBuyerID:   conv.PlatformBuyerID,
		Direction:         model.DirectionOut,
		Content:           content,
		SentAt:            now,
	}
	if _, err := s.messages().CreateIgnoreDuplicate(msg); err != nil {
		return nil, nil, err
	}
	preview := messagePreview(content)
	if conv.LastMessageAt == nil || now.After(*conv.LastMessageAt) {
		conv.LastMessageAt = &now
		conv.LastMessagePreview = preview
		_ = s.conversations().Save(conv)
	}
	if msg.ID == 0 {
		// CreateIgnoreDuplicate may not fill ID on conflict; reload not required for UI
	}
	return row, msg, nil
}

func (s *ShopService) maybeAutoReply(shop *model.CsShop, conv *model.CsConversation, inbound *model.CsMessage) {
	if inbound == nil || inbound.Direction != model.DirectionIn {
		return
	}
	if strings.HasPrefix(strings.TrimSpace(inbound.Content), "[图片]") {
		return
	}
	exists, err := s.outbound().ExistsTrigger(inbound.PlatformMessageID)
	if err != nil || exists {
		return
	}
	rules, err := s.rules().ListEnabledForShop(shop.ID)
	if err == nil {
		for i := range rules {
			rule := &rules[i]
			if !matchAutoReply(inbound.Content, rule.MatchMode, splitKeywords(rule.Keywords)) {
				continue
			}
			cool := time.Duration(rule.CooldownSec) * time.Second
			if cool <= 0 {
				cool = 45 * time.Second
			}
			recent, err := s.outbound().HasRecentAuto(conv.ID, time.Now().Add(-cool))
			if err != nil || recent {
				return
			}
			_, _, _ = s.enqueueOutbound(shop, conv, rule.ReplyText, model.ReplySourceAuto, rule.ID, inbound.PlatformMessageID)
			return
		}
	}
	s.maybeLlmReply(shop, conv, inbound)
}

func (s *ShopService) ClaimOutbound(shop *model.CsShop, limit int) ([]dto.PluginOutboundItem, error) {
	svc := s.ForTenant(shop.TenantID)
	list, err := svc.outbound().ClaimPending(shop.ID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]dto.PluginOutboundItem, 0, len(list))
	for i := range list {
		item := list[i]
		out = append(out, dto.PluginOutboundItem{
			ID:              item.ID,
			ConversationID:  item.ConversationID,
			Platform:        item.Platform,
			PlatformShopID:  item.PlatformShopID,
			PlatformBuyerID: item.PlatformBuyerID,
			BuyerName:       item.BuyerName,
			Content:         item.Content,
			Source:          item.Source,
		})
	}
	return out, nil
}

func (s *ShopService) AckOutbound(shop *model.CsShop, id uint64, ok bool, errText string) error {
	svc := s.ForTenant(shop.TenantID)
	row, err := svc.outbound().Get(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if row.ShopID != shop.ID {
		return ErrNotFound
	}
	now := time.Now()
	if ok {
		row.Status = model.OutboundSent
		row.Error = ""
		row.SentAt = &now
	} else {
		row.Status = model.OutboundFailed
		row.Error = strings.TrimSpace(errText)
		if row.Error == "" {
			row.Error = "发送失败"
		}
	}
	return svc.outbound().Save(row)
}

func toAutoReplyItem(row *model.CsAutoReplyRule) dto.AutoReplyRuleItem {
	return dto.AutoReplyRuleItem{
		ID:          row.ID,
		ShopID:      row.ShopID,
		Name:        row.Name,
		Enabled:     row.Enabled,
		MatchMode:   row.MatchMode,
		Keywords:    row.Keywords,
		ReplyText:   row.ReplyText,
		Priority:    row.Priority,
		CooldownSec: row.CooldownSec,
		CreatedAt:   formatTime(row.CreatedAt),
		UpdatedAt:   formatTime(row.UpdatedAt),
	}
}
