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
