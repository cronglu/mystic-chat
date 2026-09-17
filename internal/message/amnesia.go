package message

import (
	"fmt"
	"math/rand"
	"time"
)

// WeatheredRunes 风化残卷与消散尘埃符号
var WeatheredRunes = []rune{'·', '*', '°', '~', '.', ' ', '·', '°'}

// AmnesiaState 包含用于 TUI 渲染的褪色与风化后展示数据
type AmnesiaState struct {
	RenderedText   string        // 当前经过风化侵蚀后的文本
	TextColor      string        // 当前 ANSI 渐变色（由高亮白逐渐衰减为暗炭灰）
	Remaining      time.Duration // 剩余存活时间
	RemainingRatio float64       // 剩余时间比例 (1.0 -> 0.0)
	IsExpired      bool          // 是否已经死亡
}

// ComputeAmnesia 计算指定消息在当前时间点的衰变视觉状态
func ComputeAmnesia(text string, createdAt time.Time, ttl time.Duration, now time.Time) AmnesiaState {
	elapsed := now.Sub(createdAt)
	if elapsed >= ttl {
		return AmnesiaState{
			RenderedText:   "",
			TextColor:      "#111827",
			Remaining:      0,
			RemainingRatio: 0,
			IsExpired:      true,
		}
	}

	remaining := ttl - elapsed
	ratio := float64(remaining) / float64(ttl)
	if ratio < 0 {
		ratio = 0
	} else if ratio > 1 {
		ratio = 1
	}

	var color string
	runes := []rune(text)

	// 色彩阶梯与文字风化算法
	switch {
	case ratio >= 0.7:
		// 阶段 1：高光清晰期 (70% - 100%)
		color = "#f8fafc" // 皓月白
	case ratio >= 0.45:
		// 阶段 2：初显褪色期 (45% - 70%)
		color = "#94a3b8" // 水墨灰
	case ratio >= 0.25:
		// 阶段 3：深层墨淡期 (25% - 45%)
		color = "#64748b" // 苍云灰
	default:
		// 阶段 4：失忆解体与残卷风化期 (0% - 25%)
		color = "#475569" // 灰烬暗色

		// 字符瓦解概率：ratio 越接近 0，被风化瓦解为尘埃残符的概率越高
		// ratio = 0.25 时约 30% 字符瓦解，ratio = 0.05 时约 85% 瓦解
		erosionChance := 1.0 - (ratio / 0.25)*0.7

		// 基于消息内容本身和当前时间的种子生成伪随机风化，确保同一秒内视觉相对平稳
		erosionSeed := int64(len(runes)*1000) + now.Unix()
		rng := rand.New(rand.NewSource(erosionSeed))

		weathered := make([]rune, len(runes))
		for i, r := range runes {
			if r == ' ' || r == '\t' || r == '\n' {
				weathered[i] = r
				continue
			}

			if rng.Float64() < erosionChance {
				// 替换为风化粒子
				particle := WeatheredRunes[rng.Intn(len(WeatheredRunes))]
				weathered[i] = particle
			} else {
				weathered[i] = r
			}
		}
		runes = weathered
	}

	return AmnesiaState{
		RenderedText:   string(runes),
		TextColor:      color,
		Remaining:      remaining,
		RemainingRatio: ratio,
		IsExpired:      false,
	}
}

// FormatRemaining 格式化剩余时间显示，如 "42s" 或 "1m12s"
func FormatRemaining(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	s := int(d.Seconds())
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	return fmt.Sprintf("%dm%02ds", s/60, s%60)
}
