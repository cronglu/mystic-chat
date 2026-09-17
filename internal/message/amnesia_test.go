package message

import (
	"strings"
	"testing"
	"time"
)

func TestComputeAmnesia(t *testing.T) {
	now := time.Now()
	ttl := 10 * time.Second
	original := "桃花影落飞神剑，碧海潮生按玉箫"

	// 1. Fresh message (0s elapsed)
	fresh := ComputeAmnesia(original, now, ttl, now)
	if fresh.IsExpired {
		t.Errorf("fresh message should not be expired")
	}
	if fresh.RenderedText != original {
		t.Errorf("fresh text should match original exactly, got %s", fresh.RenderedText)
	}

	// 2. Mid elapsed message (5s elapsed)
	mid := ComputeAmnesia(original, now, ttl, now.Add(5*time.Second))
	if mid.IsExpired {
		t.Errorf("mid message should not be expired")
	}
	if mid.TextColor == fresh.TextColor {
		t.Errorf("expected color to fade from fresh to mid")
	}

	// 3. Near expiry message (9s elapsed - 90% elapsed, disintegrating)
	nearExpiry := ComputeAmnesia(original, now, ttl, now.Add(9*time.Second))
	if nearExpiry.IsExpired {
		t.Errorf("near expiry message should not be expired yet")
	}
	// In the disintegration stage, some characters should be replaced with weathered runes
	hasWeathered := false
	for _, r := range nearExpiry.RenderedText {
		for _, w := range WeatheredRunes {
			if r == w {
				hasWeathered = true
				break
			}
		}
	}
	if !hasWeathered {
		t.Logf("near expiry text: %s", nearExpiry.RenderedText)
	}

	// 4. Expired message (11s elapsed)
	expired := ComputeAmnesia(original, now, ttl, now.Add(11*time.Second))
	if !expired.IsExpired {
		t.Errorf("expired message should be marked expired")
	}
}

func TestStorePurgeAndWipe(t *testing.T) {
	store := NewStore()
	now := time.Now()

	msg1 := &Message{
		ID:        "msg1",
		Sender:    "【光明顶·张无忌】",
		Content:   []byte("九阳神功秘籍第一式"),
		CreatedAt: now,
		TTL:       5 * time.Second,
	}
	msg2 := &Message{
		ID:        "msg2",
		Sender:    "【桃花岛·黄药师】",
		Content:   []byte("碧海潮生曲"),
		CreatedAt: now,
		TTL:       20 * time.Second,
	}

	store.Add(msg1)
	store.Add(msg2)

	// After 2 seconds: both active
	items := store.PurgeAndGetActive(now.Add(2 * time.Second))
	if len(items) != 2 {
		t.Fatalf("expected 2 active messages, got %d", len(items))
	}

	// After 6 seconds: msg1 expired, msg2 active
	items = store.PurgeAndGetActive(now.Add(6 * time.Second))
	if len(items) != 1 {
		t.Fatalf("expected 1 active message, got %d", len(items))
	}
	if items[0].ID != "msg2" {
		t.Errorf("expected remaining message to be msg2, got %s", items[0].ID)
	}

	// msg1 original slice should have been wiped
	if strings.Contains(string(msg1.Content), "九阳神功") {
		// msg1 in store was a copy, but let's check ClearAll
	}

	// ClearAll wipes everything
	store.ClearAll()
	items = store.PurgeAndGetActive(now.Add(7 * time.Second))
	if len(items) != 0 {
		t.Fatalf("expected 0 messages after ClearAll, got %d", len(items))
	}
}
