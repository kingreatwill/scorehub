package tts

import "testing"

func TestBuildReceivedScoreText(t *testing.T) {
	cases := []struct {
		delta float64
		want  string
	}{
		{delta: 8, want: "收到8分"},
		{delta: 8.5, want: "收到8.5分"},
		{delta: 8.25, want: "收到8.25分"},
		{delta: 0, want: "收到分数"},
	}

	for _, tc := range cases {
		if got := BuildReceivedScoreText(tc.delta); got != tc.want {
			t.Fatalf("delta=%v got %q want %q", tc.delta, got, tc.want)
		}
	}
}
