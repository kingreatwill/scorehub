package store

import (
	"crypto/rand"
	"math"
	"math/big"
	"strings"
	"time"
)

func randomInviteCode(n int) string {
	const alphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	var b strings.Builder
	b.Grow(n)
	max := big.NewInt(int64(len(alphabet)))
	for i := 0; i < n; i++ {
		x, err := rand.Int(rand.Reader, max)
		if err != nil {
			// fallback: time-based
			b.WriteByte(alphabet[time.Now().UnixNano()%int64(len(alphabet))])
			continue
		}
		b.WriteByte(alphabet[x.Int64()])
	}
	return b.String()
}

func normalizeAmount(v float64) (float64, bool) {
	if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	rounded := math.Round(v*100) / 100
	if math.Abs(v-rounded) > 1e-6 {
		return 0, false
	}
	return rounded, true
}
