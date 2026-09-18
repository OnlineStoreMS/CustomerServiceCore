package llm

import (
	"strings"
	"testing"
)

func TestHumanizeReply_stripsAiFlavor(t *testing.T) {
	got := HumanizeReply("您好！我来帮您。根据您的描述，这个飞轮支持12速。希望对您有帮助！", 40)
	if got == "" {
		t.Fatal("empty")
	}
	for _, bad := range []string{"希望对您有帮助", "根据您的描述", "您好！"} {
		if strings.Contains(got, bad) {
			t.Fatalf("still has %q: %q", bad, got)
		}
	}
}

func TestHumanizeReply_keepsShortTalk(t *testing.T) {
	got := HumanizeReply("在的，这个支持12速", 40)
	if got != "在的，这个支持12速" {
		t.Fatalf("got %q", got)
	}
}

func TestLooksLikeStall(t *testing.T) {
	if !LooksLikeStall("我帮你看下") || !LooksLikeStall("这俩我给你对比下，稍等哈") {
		t.Fatal("should detect stall")
	}
	if LooksLikeStall("U6000更轻，6020更耐造，通勤选6020") {
		t.Fatal("real answer is not stall")
	}
}

func TestSimilarReply_repeatSkin(t *testing.T) {
	if !SimilarReply("7120是机械，7170是电变，差挺多的。", "7120是机械变速，7170是电子变速，差挺多的") {
		t.Fatal("same contrast should match")
	}
	if !SimilarReply("捷安特12速的型号挺多，你说的是哪一款？", "捷安特12速的车型挺多，你得说具体型号我才能帮你对，比如TCR、SCR这些，你指的是哪款？") {
		t.Fatal("giant 12s ask-model should match")
	}
	if !SimilarReply("在的，你说", "人工在的，你说") {
		t.Fatal("short ack should match")
	}
	if SimilarReply("7120是机械变速", "这款飞轮是12速") {
		t.Fatal("different answers should not match")
	}
}

func TestStallAlreadySaid(t *testing.T) {
	tr := "买家：U6000和6020的区别\n客服：我帮你看下\n买家：什么区别\n"
	if !stallAlreadySaid(tr) {
		t.Fatal("客服 stall should be detected")
	}
	if !looksLikeProductQuestion(tr) {
		t.Fatal("区别 should be product question")
	}
}
