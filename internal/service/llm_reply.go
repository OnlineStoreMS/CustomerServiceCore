package service

import (
	"context"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"customerservicecore/internal/dto"
	"customerservicecore/internal/model"
)

func (s *ShopService) GetLlmSetting() (*dto.LlmSettingItem, error) {
	row, err := s.repos.LlmSetting.ForTenant(s.tenant).GetOrDefault()
	if err != nil {
		return nil, err
	}
	return &dto.LlmSettingItem{
		Configured:  s.llm != nil && s.llm.Configured(),
		Enabled:     row.Enabled,
		StyleHint:   row.StyleHint,
		CooldownSec: row.CooldownSec,
		Model:       s.llmModel(),
		MaxChars:    s.llmMaxChars(),
	}, nil
}

func (s *ShopService) SaveLlmSetting(in *dto.LlmSettingInput) (*dto.LlmSettingItem, error) {
	row, err := s.repos.LlmSetting.ForTenant(s.tenant).GetOrDefault()
	if err != nil {
		return nil, err
	}
	if in.Enabled != nil {
		row.Enabled = *in.Enabled
	}
	row.StyleHint = strings.TrimSpace(in.StyleHint)
	if in.CooldownSec != nil {
		row.CooldownSec = *in.CooldownSec
	}
	if row.CooldownSec <= 0 {
		row.CooldownSec = 25
	}
	if err := s.repos.LlmSetting.ForTenant(s.tenant).Save(row); err != nil {
		return nil, err
	}
	return s.GetLlmSetting()
}

func (s *ShopService) llmModel() string {
	if s.llm == nil {
		return "deepseek-chat"
	}
	return s.llm.Model()
}

func (s *ShopService) llmMaxChars() int {
	if s.llm == nil {
		return 40
	}
	return s.llm.MaxChars()
}

func (s *ShopService) maybeLlmReply(shop *model.CsShop, conv *model.CsConversation, inbound *model.CsMessage) {
	if s.llm == nil || !s.llm.Configured() {
		return
	}
	if inbound == nil || isJunkMessageContent(inbound.Content) {
		return
	}
	content := strings.TrimSpace(inbound.Content)
	if content == "" || utf8.RuneCountInString(content) > 80 {
		return
	}
	row, err := s.repos.LlmSetting.ForTenant(s.tenant).GetOrDefault()
	if err != nil || row == nil || !row.Enabled {
		return
	}
	cool := 25
	if row.CooldownSec > 0 {
		cool = row.CooldownSec
	}
	recent, err := s.outbound().HasRecentAuto(conv.ID, time.Now().Add(-time.Duration(cool)*time.Second))
	if err != nil || recent {
		return
	}
	shopCopy, convCopy, inboundCopy, hint := *shop, *conv, *inbound, row.StyleHint
	tenant := s.tenant
	go s.runLlmReply(tenant, &shopCopy, &convCopy, &inboundCopy, hint)
}

func (s *ShopService) runLlmReply(tenantID uint64, shop *model.CsShop, conv *model.CsConversation, inbound *model.CsMessage, styleHint string) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("llm reply panic: %v", rec)
		}
	}()
	svc := s.ForTenant(tenantID)
	exists, err := svc.outbound().ExistsTrigger(inbound.PlatformMessageID)
	if err != nil || exists {
		return
	}
	recent, err := svc.messages().ListRecentByConversation(conv.ID, 6)
	if err != nil {
		return
	}
	var b strings.Builder
	for i := range recent {
		who := "客服"
		if recent[i].Direction == model.DirectionIn {
			who = "买家"
		}
		line := strings.TrimSpace(recent[i].Content)
		if line == "" || isJunkMessageContent(line) {
			continue
		}
		if utf8.RuneCountInString(line) > 60 {
			line = string([]rune(line)[:60])
		}
		b.WriteString(who)
		b.WriteString("：")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	if inbound.Content != "" && !strings.Contains(b.String(), inbound.Content) {
		b.WriteString("买家：")
		b.WriteString(strings.TrimSpace(inbound.Content))
		b.WriteByte('\n')
	}
	text, err := svc.llm.Reply(context.Background(), shop.Name, styleHint, b.String())
	if err != nil {
		log.Printf("llm reply shop=%s: %v", shop.Name, err)
		return
	}
	exists, err = svc.outbound().ExistsTrigger(inbound.PlatformMessageID)
	if err != nil || exists {
		return
	}
	_, _, _ = svc.enqueueOutbound(shop, conv, text, model.ReplySourceLlm, 0, inbound.PlatformMessageID)
}
