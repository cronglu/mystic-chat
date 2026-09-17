package identity

import (
	"strings"
	"testing"
)

func TestGenerateIdentity(t *testing.T) {
	for i := 0; i < 20; i++ {
		id := GenerateIdentity()
		if !strings.HasPrefix(id, "【") || !strings.HasSuffix(id, "】") {
			t.Errorf("expected identity wrapped in brackets, got %s", id)
		}
		if !strings.Contains(id, "·") {
			t.Errorf("expected identity containing separator '·', got %s", id)
		}
		color := GetIdentityColor(id)
		if len(color) == 0 {
			t.Errorf("expected non-empty color for identity %s", id)
		}
	}
}
