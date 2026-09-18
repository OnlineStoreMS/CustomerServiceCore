package llm

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	mdMarkRe = regexp.MustCompile("(?m)^\\s{0,3}(#{1,6}|[-*•]|\\d+\\.)\\s+")
	mdWrapRe = regexp.MustCompile("[*_`#]+")
	spaceRe  = regexp.MustCompile(`\s+`)
	aiFlavor = []string{
		"作为人工智能", "作为ai", "作为一名ai", "我是ai", "我是一个ai",
		"很高兴为您服务", "很高兴为您", "希望对您有帮助", "如有其他问题",
		"如果您还有其他问题", "感谢您的咨询", "感谢您的提问", "亲爱的用户",
		"亲爱的顾客", "您好！", "你好！我来帮您", "我来为您解答",
		"根据您的描述", "根据您提供的信息", "综上所述", "总的来说",
		"建议您可以", "您可以考虑", "请问还有什么可以帮您",
	}
)

// HumanizeReply 把模型输出收成飞鸽里能直接发的短口语。
func HumanizeReply(raw string, maxChars int) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if i := strings.Index(s, "</think>"); i >= 0 {
		s = strings.TrimSpace(s[i+len("</think>"):])
	}
	s = strings.Trim(s, "\"“”'‘’")
	s = mdMarkRe.ReplaceAllString(s, "")
	s = mdWrapRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\r", "\n")
	parts := strings.Split(s, "\n")
	keep := make([]string, 0, 2)
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		keep = append(keep, p)
		if len(keep) >= 2 {
			break
		}
	}
	s = strings.Join(keep, "")
	s = spaceRe.ReplaceAllString(s, "")
	lower := strings.ToLower(s)
	for _, bad := range aiFlavor {
		if strings.Contains(lower, bad) {
			s = stripIgnoreCase(s, bad)
			lower = strings.ToLower(s)
		}
	}
	s = strings.TrimSpace(s)
	s = strings.TrimLeft(s, "，,。.!！?？；;")
	if maxChars <= 0 {
		maxChars = 40
	}
	if utf8.RuneCountInString(s) > maxChars {
		rs := []rune(s)
		s = string(rs[:maxChars])
		s = strings.TrimRight(s, "，,、；; ")
	}
	if utf8.RuneCountInString(s) < 2 {
		return ""
	}
	return s
}

func LooksLikeStall(s string) bool {
	t := strings.TrimSpace(s)
	if t == "" {
		return false
	}
	return strings.Contains(t, "帮你看") || strings.Contains(t, "我去看") ||
		strings.Contains(t, "稍等") || strings.Contains(t, "等一下") ||
		strings.Contains(t, "对比下") || strings.Contains(t, "对比一下") ||
		strings.Contains(t, "问一下") || strings.Contains(t, "看一下库存")
}

func stallAlreadySaid(transcript string) bool {
	for _, line := range strings.Split(transcript, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "客服：") && LooksLikeStall(strings.TrimPrefix(line, "客服：")) {
			return true
		}
	}
	return false
}

func looksLikeProductQuestion(transcript string) bool {
	t := transcript
	return strings.Contains(t, "区别") || strings.Contains(t, "规格") ||
		strings.Contains(t, "多少速") || strings.Contains(t, "适不适合") ||
		strings.Contains(t, "哪个好") || strings.Contains(t, "差在")
}

func stripIgnoreCase(s, needle string) string {
	ls, ln := strings.ToLower(s), strings.ToLower(needle)
	idx := strings.Index(ls, ln)
	if idx < 0 {
		return s
	}
	return strings.TrimSpace(s[:idx] + s[idx+len(needle):])
}
