package service

import (
	"context"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"customerservicecore/internal/dto"
	"customerservicecore/internal/model"
	"customerservicecore/internal/pkg/llm"
)

func (s *ShopService) GetLlmSetting() (*dto.LlmSettingItem, error) {
	row, err := s.repos.LlmSetting.ForTenant(s.tenant).GetOrDefault()
	if err != nil {
		return nil, err
	}
	return s.toLlmSettingItem(s.fillLlmDefaults(row)), nil
}

func (s *ShopService) SaveLlmSetting(in *dto.LlmSettingInput) (*dto.LlmSettingItem, error) {
	row, err := s.repos.LlmSetting.ForTenant(s.tenant).GetOrDefault()
	if err != nil {
		return nil, err
	}
	row = s.fillLlmDefaults(row)
	if in.Enabled != nil {
		row.Enabled = *in.Enabled
	}
	if in.StyleHint != nil {
		row.StyleHint = strings.TrimSpace(*in.StyleHint)
	}
	if in.CooldownSec != nil {
		row.CooldownSec = *in.CooldownSec
	}
	if in.SystemPrompt != nil {
		row.SystemPrompt = strings.TrimSpace(*in.SystemPrompt)
	}
	if in.Model != nil {
		row.Model = strings.TrimSpace(*in.Model)
	}
	if in.MaxChars != nil {
		row.MaxChars = *in.MaxChars
	}
	if in.MaxTokens != nil {
		row.MaxTokens = *in.MaxTokens
	}
	if in.TimeoutSec != nil {
		row.TimeoutSec = *in.TimeoutSec
	}
	if in.Temperature != nil {
		row.Temperature = *in.Temperature
	}
	if in.ThinkingEnabled != nil {
		row.ThinkingEnabled = *in.ThinkingEnabled
	}
	if in.HistoryCount != nil {
		row.HistoryCount = *in.HistoryCount
	}
	if in.InboundMaxChars != nil {
		row.InboundMaxChars = *in.InboundMaxChars
	}
	if in.UseProductContext != nil {
		row.UseProductContext = *in.UseProductContext
	}
	if in.RetryStall != nil {
		row.RetryStall = *in.RetryStall
	}
	row = s.fillLlmDefaults(row)
	row = clampLlmSetting(row)
	if err := s.repos.LlmSetting.ForTenant(s.tenant).Save(row); err != nil {
		return nil, err
	}
	return s.GetLlmSetting()
}

func (s *ShopService) toLlmSettingItem(row *model.CsLlmSetting) *dto.LlmSettingItem {
	return &dto.LlmSettingItem{
		Configured:          s.llm != nil && s.llm.Configured(),
		Enabled:             row.Enabled,
		StyleHint:           row.StyleHint,
		CooldownSec:         row.CooldownSec,
		SystemPrompt:        row.SystemPrompt,
		DefaultSystemPrompt: llm.DefaultSystemPrompt,
		Model:               row.Model,
		MaxChars:            row.MaxChars,
		MaxTokens:           row.MaxTokens,
		TimeoutSec:          row.TimeoutSec,
		Temperature:         row.Temperature,
		ThinkingEnabled:     row.ThinkingEnabled,
		HistoryCount:        row.HistoryCount,
		InboundMaxChars:     row.InboundMaxChars,
		UseProductContext:   row.UseProductContext,
		RetryStall:          row.RetryStall,
	}
}

func (s *ShopService) fillLlmDefaults(row *model.CsLlmSetting) *model.CsLlmSetting {
	if row == nil {
		row = &model.CsLlmSetting{}
	}
	if row.CooldownSec <= 0 {
		row.CooldownSec = 25
	}
	if strings.TrimSpace(row.SystemPrompt) == "" {
		row.SystemPrompt = llm.DefaultSystemPrompt
		row.UseProductContext = true
		row.RetryStall = true
		row.ThinkingEnabled = false
	}
	if strings.TrimSpace(row.Model) == "" {
		row.Model = s.llmModel()
	}
	if row.MaxChars <= 0 {
		row.MaxChars = s.llmMaxChars()
	}
	if row.MaxTokens <= 0 {
		row.MaxTokens = 220
	}
	if row.TimeoutSec <= 0 {
		row.TimeoutSec = 12
	}
	if row.Temperature <= 0 {
		row.Temperature = 0.35
	}
	if row.HistoryCount <= 0 {
		row.HistoryCount = 10
	}
	if row.InboundMaxChars <= 0 {
		row.InboundMaxChars = 80
	}
	return clampLlmSetting(row)
}

func clampLlmSetting(row *model.CsLlmSetting) *model.CsLlmSetting {
	if row == nil {
		return row
	}
	row.CooldownSec = clampInt(row.CooldownSec, 10, 180)
	row.MaxChars = clampInt(row.MaxChars, 20, 200)
	row.MaxTokens = clampInt(row.MaxTokens, 32, 800)
	row.TimeoutSec = clampInt(row.TimeoutSec, 5, 60)
	row.HistoryCount = clampInt(row.HistoryCount, 2, 30)
	row.InboundMaxChars = clampInt(row.InboundMaxChars, 20, 200)
	if row.Temperature < 0.05 {
		row.Temperature = 0.05
	}
	if row.Temperature > 1.2 {
		row.Temperature = 1.2
	}
	if len(row.Model) > 64 {
		row.Model = row.Model[:64]
	}
	if len(row.StyleHint) > 256 {
		row.StyleHint = row.StyleHint[:256]
	}
	return row
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func (s *ShopService) llmModel() string {
	if s.llm == nil {
		return "deepseek-flash"
	}
	return s.llm.Model()
}

func (s *ShopService) llmMaxChars() int {
	if s.llm == nil {
		return 80
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
	row, err := s.repos.LlmSetting.ForTenant(s.tenant).GetOrDefault()
	if err != nil || row == nil || !row.Enabled {
		return
	}
	row = s.fillLlmDefaults(row)
	limit := row.InboundMaxChars
	if limit <= 0 {
		limit = 80
	}
	if content == "" || utf8.RuneCountInString(content) > limit {
		return
	}
	cool := row.CooldownSec
	if cool <= 0 {
		cool = 25
	}
	recent, err := s.outbound().HasRecentAuto(conv.ID, time.Now().Add(-time.Duration(cool)*time.Second))
	if err != nil || recent {
		return
	}
	shopCopy, convCopy, inboundCopy, settingCopy := *shop, *conv, *inbound, *row
	tenant := s.tenant
	go s.runLlmReply(tenant, &shopCopy, &convCopy, &inboundCopy, &settingCopy)
}

func (s *ShopService) runLlmReply(tenantID uint64, shop *model.CsShop, conv *model.CsConversation, inbound *model.CsMessage, setting *model.CsLlmSetting) {
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
	history := 10
	if setting != nil && setting.HistoryCount > 0 {
		history = setting.HistoryCount
	}
	recent, err := svc.messages().ListRecentByConversation(conv.ID, history)
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
		if utf8.RuneCountInString(line) > 80 {
			line = string([]rune(line)[:80])
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
	if setting == nil {
		return
	}
	productBg := conv.ProductContext
	if !setting.UseProductContext {
		productBg = ""
	}
	text, err := svc.llm.Reply(context.Background(), llm.ReplyOptions{
		ShopName:          shop.Name,
		StyleHint:         setting.StyleHint,
		ProductBackground: productBg,
		Transcript:        b.String(),
		SystemPrompt:      setting.SystemPrompt,
		Model:             setting.Model,
		MaxChars:          setting.MaxChars,
		MaxTokens:         setting.MaxTokens,
		TimeoutSec:        setting.TimeoutSec,
		Temperature:       setting.Temperature,
		Thinking:          setting.ThinkingEnabled,
		UseProductContext: setting.UseProductContext,
		RetryStall:        setting.RetryStall,
	})
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
