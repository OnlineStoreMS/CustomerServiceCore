package service

import "testing"

func TestParseVisionMessages(t *testing.T) {
	raw := "```json\n{\"messages\":[{\"direction\":\"in\",\"text\":\"在吗\"},{\"direction\":\"out\",\"text\":\"在的\"},{\"direction\":\"system\",\"text\":\"客服接入\"}]}\n```"
	got, err := ParseVisionMessages(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Text != "在吗" || got[1].Direction != "out" {
		t.Fatalf("got %#v", got)
	}
}

func TestAlignVisionAppendsTail(t *testing.T) {
	history := []visionLine{
		{Direction: "in", Text: "在吗", ID: "vis:1"},
		{Direction: "out", Text: "在的", ID: "vis:2"},
	}
	visible := []visionLine{
		{Direction: "in", Text: "在吗"},
		{Direction: "out", Text: "在的"},
		{Direction: "in", Text: "这款有货吗"},
	}
	fresh, ok := AlignVision(history, visible)
	if !ok || len(fresh) != 1 || fresh[0].Text != "这款有货吗" || fresh[0].ID == "" {
		t.Fatalf("ok=%v fresh=%#v", ok, fresh)
	}
}

func TestAlignVisionSkipsUnrelatedScreen(t *testing.T) {
	history := []visionLine{
		{Direction: "in", Text: "车把多少钱"},
		{Direction: "out", Text: "这款 299"},
		{Direction: "in", Text: "有黑色吗"},
		{Direction: "out", Text: "有的"},
	}
	visible := []visionLine{
		{Direction: "in", Text: "你好我是另一个人"},
		{Direction: "out", Text: "请问需要什么"},
	}
	fresh, ok := AlignVision(history, visible)
	if ok || len(fresh) != 0 {
		t.Fatalf("ok=%v fresh=%#v", ok, fresh)
	}
}

func TestAlignVisionExtendsCutOffBubble(t *testing.T) {
	history := []visionLine{{
		Direction: "out",
		Text:      "DA手变是碳纤维手柄，UT是铝合金，DA手感更轻脆，重量也轻一",
		ID:        "vis:cut",
	}}
	visible := []visionLine{{
		Direction: "out",
		Text:      "DA手变是碳纤维手柄，UT是铝合金，DA手感更轻脆，重量也轻一些，UT更耐用实惠",
	}}
	fresh, ok := AlignVision(history, visible)
	if !ok || len(fresh) != 1 || !fresh[0].Update || fresh[0].ID != "vis:cut" || fresh[0].Text != visible[0].Text {
		t.Fatalf("ok=%v fresh=%#v", ok, fresh)
	}
}

func TestAlignVisionKeepsLongerHistory(t *testing.T) {
	history := []visionLine{{
		Direction: "out",
		Text:      "DA手变是碳纤维手柄，UT是铝合金，DA手感更轻脆，重量也轻一些，UT更耐用实惠",
		ID:        "vis:full",
	}}
	visible := []visionLine{{
		Direction: "out",
		Text:      "DA手变是碳纤维手柄，UT是铝合金，DA手感更轻脆，重量也轻一",
	}}
	fresh, ok := AlignVision(history, visible)
	if !ok || len(fresh) != 0 {
		t.Fatalf("ok=%v fresh=%#v", ok, fresh)
	}
}

func TestAlignVisionRepeatHello(t *testing.T) {
	history := []visionLine{{Direction: "in", Text: "您好", ID: "vis:a"}}
	visible := []visionLine{{Direction: "in", Text: "您好"}, {Direction: "in", Text: "您好"}}
	fresh, ok := AlignVision(history, visible)
	if !ok || len(fresh) != 1 || fresh[0].ID == history[0].ID {
		t.Fatalf("ok=%v fresh=%#v", ok, fresh)
	}
}
