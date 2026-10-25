package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"customerservicecore/internal/dto"
	"customerservicecore/internal/model"
	"customerservicecore/internal/pkg/aimodel"
)

const visionSystemPrompt = `你是飞鸽客服聊天记录抄写员。只根据图片里中间的聊天区，从上到下逐条抄写气泡原文。
可能有多张连续翻页截图，属于同一会话：按整段从上到下抄写，重复出现的气泡只保留一次完整原文。
左侧或靠左的浅色气泡是顾客，direction 为 in。
右侧或靠右的彩色气泡是商家客服，direction 为 out。
不要概括，不要合并相邻气泡，不要补写图片里没有的字。
居中的灰色系统提示、已读、推荐回复、输入框草稿不要当成消息。
气泡旁边的灰色时间（如 14:32、昨天 14:32）写入 sentAt，能看成完整日期就写成 2006-01-02 15:04:05，只有钟点就写 15:04:05；没有时间就空字符串。
图片气泡 text 写「[图片]」。商品卡片 text 写「[商品]」加卡片标题原文。
只输出 JSON，不要 Markdown：{"messages":[{"direction":"in","text":"原文","sentAt":"14:32:00"}]}`

const visionUserPrompt = "这些图是同一会话从上往下翻页的连续截图。按时间顺序抄完整记录，重复气泡只留一次，并带上每条旁边能看到的时间。"

type visionLine struct {
	Direction string
	Text      string
	ID        string
	SentAt    string
	Update    bool
}

func (s *ShopService) TranscribeVision(ctx context.Context, shop *model.CsShop, in *dto.PluginTranscribeInput) (*dto.PluginTranscribeResult, error) {
	if s.aimodel == nil {
		return nil, fmt.Errorf("%w: 未配置 AI 大模型中心", ErrBadRequest)
	}
	imageURLs, err := visionImageURLsFrom(in.ImageBase64, in.Images)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrBadRequest, err.Error())
	}
	raw, modelName, err := s.aimodel.Vision(ctx, aimodel.VisionInput{
		TenantID:  shop.TenantID,
		System:    visionSystemPrompt,
		Prompt:    visionUserPrompt,
		ImageURLs: imageURLs,
	})
	if err != nil {
		return nil, err
	}
	visible, err := ParseVisionMessages(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrBadRequest, err.Error())
	}
	out := &dto.PluginTranscribeResult{Model: modelName, Messages: make([]dto.PluginTranscribeLine, 0, len(visible))}
	for _, line := range visible {
		out.Messages = append(out.Messages, dto.PluginTranscribeLine{
			Direction: line.Direction,
			Text:      line.Text,
			SentAt:    line.SentAt,
		})
	}
	return out, nil
}

func (s *ShopService) IngestVision(ctx context.Context, shop *model.CsShop, in *dto.PluginVisionInput) (*dto.PluginVisionResult, error) {
	if s.aimodel == nil {
		return nil, fmt.Errorf("%w: 未配置 AI 大模型中心", ErrBadRequest)
	}
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
	imageURLs, err := visionImageURLsFrom(in.ImageBase64, in.Images)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrBadRequest, err.Error())
	}
	buyerName := normalizeBuyerName(in.BuyerName)
	if isJunkBuyerName(buyerName) {
		return nil, fmt.Errorf("%w: 缺少顾客昵称", ErrBadRequest)
	}

	raw, modelName, err := s.aimodel.Vision(ctx, aimodel.VisionInput{
		TenantID:  shop.TenantID,
		System:    visionSystemPrompt,
		Prompt:    visionUserPrompt,
		ImageURLs: imageURLs,
	})
	if err != nil {
		return nil, err
	}
	visible, err := ParseVisionMessages(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrBadRequest, err.Error())
	}

	svc := s.ForTenant(shop.TenantID)
	buyerID := "name:" + buyerName
	now := time.Now()
	conv, err := svc.ensureConversation(shop, platform, platformShopID, buyerID, buyerName, buyerID, now, "")
	if err != nil {
		return nil, err
	}
	historyRows, err := svc.messages().ListRecentByConversation(conv.ID, 80)
	if err != nil {
		return nil, err
	}
	var history []visionLine
	for _, row := range historyRows {
		history = append(history, visionLine{
			Direction: row.Direction,
			Text:      row.Content,
			ID:        row.PlatformMessageID,
			SentAt:    formatTime(row.SentAt),
		})
	}
	visible = attachHintTimes(visible, visionHintsFromDOM(in.DomMessages))
	fresh, aligned := AlignVision(history, visible)
	out := &dto.PluginVisionResult{Visible: len(visible), Model: modelName, Aligned: aligned}
	if !aligned {
		return out, nil
	}
	live := !in.Baseline && len(history) > 0
	capturedAt := parseSentAt(in.CapturedAt)
	for i, line := range fresh {
		if line.Update {
			if err := svc.messages().UpdateContent(line.ID, line.Text); err != nil {
				return nil, err
			}
			out.Accepted++
			if conv.LastMessagePreview != "" && strings.HasPrefix(normVisionText(line.Text), normVisionText(conv.LastMessagePreview)) {
				conv.LastMessagePreview = messagePreview(line.Text)
				_ = svc.conversations().Save(conv)
			}
			continue
		}
		sentAt := parseVisionSentAt(line.SentAt, capturedAt)
		if sentAt.IsZero() {
			sentAt = now.Add(time.Duration(i) * time.Millisecond)
		}
		rawJSON, _ := json.Marshal(map[string]string{"source": "vision", "model": modelName})
		msg := &model.CsMessage{
			ShopID:            shop.ID,
			ConversationID:    conv.ID,
			Platform:          platform,
			PlatformMessageID: line.ID,
			PlatformShopID:    platformShopID,
			PlatformBuyerID:   buyerID,
			Direction:         line.Direction,
			Content:           line.Text,
			SentAt:            sentAt,
			RawJSON:           string(rawJSON),
		}
		created, err := svc.messages().CreateIgnoreDuplicate(msg)
		if err != nil {
			return nil, err
		}
		if !created {
			out.Skipped++
			continue
		}
		out.Accepted++
		preview := messagePreview(line.Text)
		if conv.LastMessageAt == nil || sentAt.After(*conv.LastMessageAt) {
			conv.LastMessageAt = &sentAt
			conv.LastMessagePreview = preview
			conv.BuyerName = buyerName
			_ = svc.conversations().Save(conv)
		}
		if line.Direction == model.DirectionIn {
			out.Inbound++
			if live && !strings.HasPrefix(line.Text, "[") {
				svc.maybeAutoReply(shop, conv, msg, true)
			}
		}
	}
	return out, nil
}

func visionImageURLsFrom(primary string, images []string) ([]string, error) {
	raws := images
	if len(raws) <= 1 {
		raws = []string{primary}
	}
	if len(raws) > 8 {
		raws = raws[:8]
	}
	urls := make([]string, 0, len(raws))
	for _, raw := range raws {
		url, err := visionDataURL(raw)
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("缺少聊天截图")
	}
	return urls, nil
}

func visionDataURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("缺少聊天截图")
	}
	payload := raw
	if i := strings.Index(raw, ","); i >= 0 && strings.Contains(raw[:i], "base64") {
		payload = raw[i+1:]
	}
	buf, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		buf, err = base64.RawStdEncoding.DecodeString(payload)
	}
	if err != nil || len(buf) < 32 {
		return "", fmt.Errorf("聊天截图无法解析")
	}
	if len(buf) > 4<<20 {
		return "", fmt.Errorf("聊天截图过大")
	}
	if strings.HasPrefix(raw, "data:image/") {
		return raw, nil
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf), nil
}

func ParseVisionMessages(raw string) ([]visionLine, error) {
	text := strings.TrimSpace(raw)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("识图结果不是 JSON")
	}
	var body struct {
		Messages []struct {
			Direction string `json:"direction"`
			Text      string `json:"text"`
			SentAt    string `json:"sentAt"`
		} `json:"messages"`
	}
	if err := json.Unmarshal([]byte(text[start:end+1]), &body); err != nil {
		return nil, fmt.Errorf("识图结果无法解析")
	}
	out := make([]visionLine, 0, len(body.Messages))
	for _, item := range body.Messages {
		dir := strings.TrimSpace(strings.ToLower(item.Direction))
		if dir != model.DirectionIn && dir != model.DirectionOut {
			continue
		}
		content := strings.TrimSpace(item.Text)
		content = strings.Join(strings.Fields(content), " ")
		if content == "" || isJunkMessageContent(content) {
			continue
		}
		if utf8.RuneCountInString(content) > 800 {
			content = string([]rune(content)[:800])
		}
		out = append(out, visionLine{Direction: dir, Text: content, SentAt: strings.TrimSpace(item.SentAt)})
		if len(out) >= 80 {
			break
		}
	}
	return out, nil
}

// AlignVision 把当前画面和已入库的识图记录按顺序对齐。对不上时不入库，避免整段重复。
func AlignVision(history, visible []visionLine) ([]visionLine, bool) {
	if len(visible) == 0 {
		return nil, true
	}
	if len(history) == 0 {
		return stampVisionIDs(nil, visible), true
	}
	bestStart, bestScore := 0, -1
	for start := 0; start <= len(history); start++ {
		score := 0
		h := start
		for _, v := range visible {
			if h < len(history) && history[h].Direction == v.Direction && visionMatch(history[h].Text, v.Text) {
				score++
				h++
			}
		}
		if score > bestScore || (score == bestScore && start > bestStart) {
			bestScore = score
			bestStart = start
		}
	}
	if bestScore == 0 && len(history) >= 4 {
		return nil, false
	}
	h := bestStart
	var fresh []visionLine
	var pending []visionLine
	for _, v := range visible {
		if h < len(history) && history[h].Direction == v.Direction && visionMatch(history[h].Text, v.Text) {
			if longer := longerVisionText(history[h].Text, v.Text); longer != "" {
				fresh = append(fresh, visionLine{
					Direction: history[h].Direction,
					Text:      longer,
					ID:        history[h].ID,
					SentAt:    firstNonEmpty(v.SentAt, history[h].SentAt),
					Update:    true,
				})
			}
			h++
			continue
		}
		pending = append(pending, v)
	}
	return append(fresh, stampVisionIDs(history, pending)...), true
}

func visionMatch(have, seen string) bool {
	h, s := normVisionText(have), normVisionText(seen)
	if h == s {
		return true
	}
	hr, sr := utf8.RuneCountInString(h), utf8.RuneCountInString(s)
	if hr >= 4 && sr >= 4 && (strings.HasPrefix(s, h) || strings.HasPrefix(h, s)) {
		return true
	}
	return sameVisionText(have, seen)
}

func longerVisionText(have, seen string) string {
	h, s := normVisionText(have), normVisionText(seen)
	if utf8.RuneCountInString(h) >= 4 && utf8.RuneCountInString(s) > utf8.RuneCountInString(h) && strings.HasPrefix(s, h) {
		return seen
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func visionHintsFromDOM(items []dto.PluginVisionHint) []visionLine {
	out := make([]visionLine, 0, len(items))
	for _, item := range items {
		dir := strings.TrimSpace(strings.ToLower(item.Direction))
		if dir != model.DirectionIn && dir != model.DirectionOut {
			continue
		}
		text := strings.Join(strings.Fields(strings.TrimSpace(item.Text)), " ")
		if text == "" {
			continue
		}
		out = append(out, visionLine{Direction: dir, Text: text, SentAt: formatHintSentAt(item.SentAt)})
	}
	return out
}

func formatHintSentAt(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
		return ""
	}
	return formatTime(parseSentAt(v))
}

func attachHintTimes(visible, hints []visionLine) []visionLine {
	if len(visible) == 0 || len(hints) == 0 {
		return visible
	}
	bestStart, bestScore := 0, -1
	for start := 0; start <= len(hints); start++ {
		score := 0
		h := start
		for _, v := range visible {
			if h < len(hints) && hints[h].Direction == v.Direction && visionMatch(hints[h].Text, v.Text) {
				score++
				h++
			}
		}
		if score > bestScore || (score == bestScore && start > bestStart) {
			bestScore = score
			bestStart = start
		}
	}
	last := ""
	for i := 0; i < bestStart && i < len(hints); i++ {
		if hints[i].SentAt != "" {
			last = hints[i].SentAt
		}
	}
	out := append([]visionLine(nil), visible...)
	h := bestStart
	for i, v := range visible {
		if h < len(hints) && hints[h].Direction == v.Direction && visionMatch(hints[h].Text, v.Text) {
			if hints[h].SentAt != "" {
				out[i].SentAt = hints[h].SentAt
				last = hints[h].SentAt
			} else if out[i].SentAt == "" {
				out[i].SentAt = last
			}
			h++
			continue
		}
		if out[i].SentAt == "" {
			out[i].SentAt = last
		}
	}
	return out
}

func parseVisionSentAt(raw string, capturedAt time.Time) time.Time {
	s := strings.TrimSpace(raw)
	if s == "" {
		return capturedAt
	}
	if clock, ok := parseClockOnly(s); ok && !parseHasDate(s) {
		day := time.Now()
		if !capturedAt.IsZero() {
			day = capturedAt
		}
		return time.Date(day.Year(), day.Month(), day.Day(), clock.Hour(), clock.Minute(), clock.Second(), 0, time.Local)
	}
	return parseSentAt(s)
}

func parseHasDate(s string) bool {
	return strings.Contains(s, "-") || strings.Contains(s, "/") || strings.Contains(s, "年")
}

func parseClockOnly(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "昨天")
	s = strings.TrimPrefix(s, "今天")
	s = strings.TrimSpace(s)
	for _, layout := range []string{"15:04:05", "15:04"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func stampVisionIDs(history, fresh []visionLine) []visionLine {
	occ := map[string]int{}
	for _, row := range history {
		occ[row.Direction+"|"+normVisionText(row.Text)]++
	}
	out := make([]visionLine, 0, len(fresh))
	for _, row := range fresh {
		key := row.Direction + "|" + normVisionText(row.Text)
		occ[key]++
		sum := sha256.Sum256([]byte(key + "|" + fmt.Sprint(occ[key])))
		row.ID = "vis:" + hex.EncodeToString(sum[:12])
		out = append(out, row)
	}
	return out
}

func sameVisionText(a, b string) bool {
	a, b = normVisionText(a), normVisionText(b)
	if a == b {
		return true
	}
	ar, br := utf8.RuneCountInString(a), utf8.RuneCountInString(b)
	if ar < 4 || br < 4 {
		return false
	}
	shorter, longer := a, b
	shortN, longN := ar, br
	if br < ar {
		shorter, longer = b, a
		shortN, longN = br, ar
	}
	if !strings.Contains(longer, shorter) {
		return false
	}
	return float64(shortN)/float64(longN) >= 0.85
}

func normVisionText(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), "")
}
