package bot

import (
	"testing"
	"time"
)

func TestPluralize(t *testing.T) {
	tests := []struct {
		n    int
		one  string
		two  string
		five string
		want string
	}{
		{1, "день", "дня", "дней", "день"},
		{2, "день", "дня", "дней", "дня"},
		{5, "день", "дня", "дней", "дней"},
		{11, "день", "дня", "дней", "дней"},
		{21, "день", "дня", "дней", "день"},
		{22, "день", "дня", "дней", "дня"},
		{25, "день", "дня", "дней", "дней"},
	}

	for _, tt := range tests {
		got := pluralize(tt.n, tt.one, tt.two, tt.five)
		if got != tt.want {
			t.Errorf("pluralize(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestFormatTimeLeft(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{
			name: "negative duration",
			d:    -10 * time.Minute,
			want: "совсем скоро",
		},
		{
			name: "minutes only",
			d:    45 * time.Minute,
			want: "45 минут",
		},
		{
			name: "hours and minutes",
			d:    3*time.Hour + 12*time.Minute,
			want: "3 часа, 12 минут",
		},
		{
			name: "days, hours and minutes",
			d:    2*24*time.Hour + 5*time.Hour + 1*time.Minute,
			want: "2 дня, 5 часов, 1 минуту",
		},
		{
			name: "exactly 1 day",
			d:    24 * time.Hour,
			want: "1 день",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTimeLeft(tt.d)
			if got != tt.want {
				t.Errorf("formatTimeLeft() = %q, want %q", got, tt.want)
			}
		})
	}
}
