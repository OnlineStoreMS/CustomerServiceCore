package service

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"customerservicecore/internal/model"
)

var (
	buyerSpaceRe  = regexp.MustCompile(`\s+`)
	buyerNoteRe   = regexp.MustCompile(`\s*添加备注.*$`)
	buyerMoreRe   = regexp.MustCompile(`\s*更多.*$`)
	buyerBadgeRe  = regexp.MustCompile(`\s*\(\d+\)\s*$`)
	buyerNameIDRe = regexp.MustCompile(`^(name|uid):`)
)

var buyerChromeExact = map[string]struct{}{
	"默认分组": {}, "添加备注": {}, "更多": {}, "备注": {}, "当前会话": {},
	"最近联系": {}, "平台消息": {}, "快捷短语": {}, "unknown": {},
	"暂无会话": {}, "请选择会话": {}, "会话搜索": {}, "商家后台": {},
}

func normalizeBuyerName(raw string) string {
	s := strings.TrimSpace(raw)
	s = buyerNameIDRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "重复来访", " ")
	s = strings.ReplaceAll(s, "进线", " ")
	s = regexp.MustCompile(`\b\d{1,2}:\d{2}(:\d{2})?\b`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`(?i)\b\d{1,2}\s*s\b`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\d{1,2}\s*秒`).ReplaceAllString(s, " ")
	s = buyerSpaceRe.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	s = buyerNoteRe.ReplaceAllString(s, "")
	s = buyerMoreRe.ReplaceAllString(s, "")
	s = buyerBadgeRe.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "添加备注")
	s = strings.TrimSpace(s)
	if fields := strings.Fields(s); len(fields) > 0 {
		s = fields[0]
	}
	return s
}

func isJunkBuyerName(raw string) bool {
	n := normalizeBuyerName(raw)
	if n == "" {
		return true
	}
	if strings.HasPrefix(n, "默认分组") {
		return true
	}
	if _, ok := buyerChromeExact[n]; ok {
		return true
	}
	trimmed := strings.TrimSpace(raw)
	trimmed = buyerNameIDRe.ReplaceAllString(trimmed, "")
	if strings.HasSuffix(trimmed, "备注") && !strings.Contains(trimmed, "添加备注") {
		base := strings.TrimSpace(strings.TrimSuffix(n, "备注"))
		if utf8.RuneCountInString(base) <= 8 {
			return true
		}
	}
	return false
}

func normalizeBuyerIdentity(buyerID, buyerName string) (id, name string, ok bool) {
	name = normalizeBuyerName(buyerName)
	if name == "" {
		name = normalizeBuyerName(buyerID)
	}
	if isJunkBuyerName(name) || isJunkBuyerName(buyerName) || isJunkBuyerName(buyerID) {
		return "", "", false
	}
	id = strings.TrimSpace(buyerID)
	if strings.HasPrefix(id, "uid:") && len(id) > 4 {
		return id, name, true
	}
	return "name:" + name, name, true
}

func conversationMergeKey(c *model.CsConversation) (key string, junk bool) {
	name := normalizeBuyerName(c.BuyerName)
	if name == "" {
		name = normalizeBuyerName(c.PlatformBuyerID)
	}
	if name == "" || isJunkBuyerName(name) {
		return name, true
	}
	return name, false
}

func conversationGroupKey(c *model.CsConversation) string {
	name, junk := conversationMergeKey(c)
	if junk || name == "" {
		return ""
	}
	return fmt.Sprintf("%d|%s|%s|%s", c.ShopID, strings.ToLower(c.Platform), strings.ToLower(c.PlatformShopID), name)
}

func conversationCanonicalScore(c *model.CsConversation) int {
	n, junk := conversationMergeKey(c)
	if junk {
		return -100
	}
	score := 0
	if c.PlatformBuyerID == "name:"+n {
		score += 6
	}
	if c.BuyerName == n {
		score += 3
	}
	if !strings.Contains(c.BuyerName, "备注") {
		score += 2
	}
	if !strings.Contains(c.BuyerName, "分组") {
		score += 2
	}
	return score
}

var junkPriceRe = regexp.MustCompile(`^(¥\s*[\d,.]+|[\d,.]+\s*元)(\s*\(\d+\s*次\))?$|^[\d,.]+\s*\(\d+\s*次\)$`)

func isJunkMessageContent(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	if strings.Contains(s, "店铺消费") || strings.Contains(s, "客单价") || strings.Contains(s, "商品详情") || strings.Contains(s, "重复来访") {
		return true
	}
	if strings.Contains(s, "用户正在查看商品") || strings.Contains(s, "电商小助手") || strings.Contains(s, "计算价格") || strings.Contains(s, "邀请下单") || strings.Contains(s, "规格/属性") {
		return true
	}
	if matched, _ := regexp.MatchString(`客服.{0,12}接入`, s); matched {
		return true
	}
	if strings.HasPrefix(s, "抖音-") {
		return true
	}
	return junkPriceRe.MatchString(s)
}
