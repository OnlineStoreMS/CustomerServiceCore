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
左侧或靠左的浅色气泡是顾客，direction 为 in。
右侧或靠右的彩色气泡是商家客服，direction 为 out。
不要概括，不要合并相邻气泡，不要补写图片里没有的字。
居中的灰色系统提示、时间、已读、推荐回复、输入框草稿不要输出。
图片气泡 text 写「[图片]」。商品卡片 text 写「[商品]」加卡片标题原文。
只输出 JSON，不要 Markdown：{"messages":[{"direction":"in","text":"原文"}]}`

const visionUserPrompt = "按从上到下的顺序，把这张飞鸽会话截图里的顾客消息和商家回复逐条抄下来。"

type visionLine struct {
	Direction string
	Text      string
	ID        string
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
	imageURL, err := visionDataURL(in.ImageBase64)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrBadRequest, err.Error())
	}
	buyerName := normalizeBuyerName(in.BuyerName)
	if isJunkBuyerName(buyerName) {
		return nil, fmt.Errorf("%w: 缺少顾客昵称", ErrBadRequest)
	}

	raw, modelName, err := s.aimodel.Vision(ctx, aimodel.VisionInput{
		TenantID: shop.TenantID,
		System:   visionSystemPrompt,
		Prompt:   visionUserPrompt,
		ImageURL: imageURL,
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
		if !strings.HasPrefix(row.PlatformMessageID, "vis:") {
			continue
		}
		history = append(history, visionLine{Direction: row.Direction, Text: row.Content, ID: row.PlatformMessageID})
	}
	if len(history) == 0 {
		if err := svc.messages().DeleteNonVision(conv.ID); err != nil {
			return nil, err
		}
	}
	fresh, aligned := AlignVision(history, visible)
	out := &dto.PluginVisionResult{Visible: len(visible), Model: modelName, Aligned: aligned}
	if !aligned {
		return out, nil
	}
	live := !in.Baseline && len(history) > 0
	for i, line := range fresh {
		sentAt := now.Add(time.Duration(i) * time.Millisecond)
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
		out = append(out, visionLine{Direction: dir, Text: content})
		if len(out) >= 40 {
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
			if h < len(history) && history[h].Direction == v.Direction && sameVisionText(history[h].Text, v.Text) {
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
	for _, v := range visible {
		if h < len(history) && history[h].Direction == v.Direction && sameVisionText(history[h].Text, v.Text) {
			h++
			continue
		}
		fresh = append(fresh, v)
	}
	return stampVisionIDs(history, fresh), true
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
