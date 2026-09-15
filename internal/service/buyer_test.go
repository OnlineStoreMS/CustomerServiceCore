package service

import (
	"testing"

	"customerservicecore/internal/model"
)

func TestNormalizeBuyerName_stripsVisitChrome(t *testing.T) {
	cases := map[string]string{
		"杭州觅景单车 重复来访 03:01":      "杭州觅景单车",
		"杭州觅景单车 重复来访 6秒":         "杭州觅景单车",
		"杭州觅景单车 重复来访 02:52 在的":   "杭州觅景单车",
		"name:杭州觅景单车 重复来访 03:01": "杭州觅景单车",
		"觅选美妆 21秒":               "觅选美妆",
		"觅选美妆 03:04 在的":          "觅选美妆",
		"觅选美妆":                   "觅选美妆",
	}
	for in, want := range cases {
		if got := normalizeBuyerName(in); got != want {
			t.Fatalf("normalizeBuyerName(%q)=%q want %q", in, got, want)
		}
	}
}

func TestMatchAutoReply_exactAndContains(t *testing.T) {
	kws := splitKeywords("好的,谢谢,谢谢老板,收到,ok")
	if !matchAutoReply("好的", model.MatchExact, kws) {
		t.Fatal("exact 好的")
	}
	if !matchAutoReply("谢谢！", model.MatchExact, kws) {
		t.Fatal("exact 谢谢 with punct")
	}
	if matchAutoReply("这个飞轮支持12速吗", model.MatchExact, kws) {
		t.Fatal("should not match question")
	}
	if !matchAutoReply("好的谢谢老板", model.MatchContains, kws) {
		t.Fatal("contains 谢谢")
	}
	if !matchAutoReply("好的 谢谢", model.MatchExact, splitKeywords("好的,好的谢谢,谢谢")) {
		t.Fatal("exact 好的 谢谢 after normalize")
	}
}

func TestIsJunkMessageContent_feigeChrome(t *testing.T) {
	for _, s := range []string{"收起", "消息来源", "发送方式 商家配置发送", "用户超时未回复，系统关闭会话", "功能路径 飞鸽-客服管理"} {
		if !isJunkMessageContent(s) {
			t.Fatalf("expected junk: %q", s)
		}
	}
	if isJunkMessageContent("好的") || isJunkMessageContent("谢谢") {
		t.Fatal("buyer greetings must not be junk")
	}
}

func TestConversationMergeKey_sameBuyer(t *testing.T) {
	rows := []*model.CsConversation{
		{ShopID: 1, Platform: "doudian", PlatformShopID: "177987746", BuyerName: "杭州觅景单车 重复来访 6秒", PlatformBuyerID: "name:杭州觅景单车 重复来访 6秒"},
		{ShopID: 1, Platform: "doudian", PlatformShopID: "177987746", BuyerName: "杭州觅景单车", PlatformBuyerID: "name:杭州觅景单车"},
		{ShopID: 1, Platform: "doudian", PlatformShopID: "177987746", BuyerName: "杭州觅景单车 重复来访 03:01", PlatformBuyerID: "name:杭州觅景单车 重复来访 03:01"},
	}
	var keys []string
	for _, row := range rows {
		key, junk := conversationMergeKey(row)
		if junk || key != "杭州觅景单车" {
			t.Fatalf("merge key %+v => %q junk=%v", row, key, junk)
		}
		if gk := conversationGroupKey(row); gk != conversationGroupKey(rows[0]) {
			t.Fatalf("group key mismatch %q vs %q", gk, conversationGroupKey(rows[0]))
		}
		keys = append(keys, key)
	}
	_ = keys
}
