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

func TestStallAlreadySaid(t *testing.T) {
	tr := "买家：U6000和6020的区别\n客服：我帮你看下\n买家：什么区别\n"
	if !stallAlreadySaid(tr) {
		t.Fatal("客服 stall should be detected")
	}
	if !looksLikeProductQuestion(tr) {
		t.Fatal("区别 should be product question")
	}
}
