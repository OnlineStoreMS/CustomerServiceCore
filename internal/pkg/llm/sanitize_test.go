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

func TestHumanizeReply_truncates(t *testing.T) {
	got := HumanizeReply("这是一段很长很长的客服解释用来测试截断是不是生效了还会继续写下去", 12)
	if len([]rune(got)) > 12 {
		t.Fatalf("too long: %q", got)
	}
}
