package parser

import (
	"testing"
	"time"
)

func TestExtractDate(t *testing.T) {
	// Set mock time: Wednesday, May 20, 2026 14:32:00 MSK
	mockNow := time.Date(2026, time.May, 20, 14, 32, 0, 0, msk)
	timeNow = func() time.Time {
		return mockNow
	}
	defer func() {
		timeNow = time.Now
	}()

	tests := []struct {
		name        string
		text        string
		wantYear    int
		wantMonth   time.Month
		wantDay     int
		wantHour    int
		wantMin     int
		wantHasTime bool
		wantNil     bool
	}{
		{
			name:        "10 июня (this year, since June 10 is after May 20)",
			text:        "Розыгрыш 10 июня",
			wantYear:    2026,
			wantMonth:   time.June,
			wantDay:     10,
			wantHour:    0,
			wantMin:     0,
			wantHasTime: false,
		},
		{
			name:        "10 мая (next year, since May 10 is before May 20)",
			text:        "Розыгрыш 10 мая",
			wantYear:    2027,
			wantMonth:   time.May,
			wantDay:     10,
			wantHour:    0,
			wantMin:     0,
			wantHasTime: false,
		},
		{
			name:        "в пятницу (22.05) в 20:00",
			text:        "в пятницу (22.05) в 20:00",
			wantYear:    2026,
			wantMonth:   time.May,
			wantDay:     22,
			wantHour:    20,
			wantMin:     0,
			wantHasTime: true,
		},
		{
			name:        "в пятницу в 20:00 (this Friday, May 22)",
			text:        "в пятницу в 20:00",
			wantYear:    2026,
			wantMonth:   time.May,
			wantDay:     22,
			wantHour:    20,
			wantMin:     0,
			wantHasTime: true,
		},
		{
			name:        "в пятницу (without time)",
			text:        "в пятницу",
			wantYear:    2026,
			wantMonth:   time.May,
			wantDay:     22,
			wantHour:    0,
			wantMin:     0,
			wantHasTime: false,
		},
		{
			name:        "22.05 (this year)",
			text:        "Итоги 22.05",
			wantYear:    2026,
			wantMonth:   time.May,
			wantDay:     22,
			wantHour:    0,
			wantMin:     0,
			wantHasTime: false,
		},
		{
			name:        "15.05 (next year)",
			text:        "Итоги 15.05",
			wantYear:    2027,
			wantMonth:   time.May,
			wantDay:     15,
			wantHour:    0,
			wantMin:     0,
			wantHasTime: false,
		},
		{
			name:    "invalid text",
			text:    "просто какой-то текст без дат",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, hasTime := extractDate(tt.text)
			if tt.wantNil {
				if got != nil {
					t.Errorf("extractDate() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("extractDate() = nil, want non-nil")
			}
			tval := got.In(msk)
			if tval.Year() != tt.wantYear || tval.Month() != tt.wantMonth || tval.Day() != tt.wantDay || tval.Hour() != tt.wantHour || tval.Minute() != tt.wantMin || hasTime != tt.wantHasTime {
				t.Errorf("extractDate() = (%v, %v), want (%d-%02d-%02d %02d:%02d, %v)",
					tval, hasTime, tt.wantYear, tt.wantMonth, tt.wantDay, tt.wantHour, tt.wantMin, tt.wantHasTime)
			}
		})
	}
}

func TestGetNextWeekdayDate_SameDay(t *testing.T) {
	// Wednesday, May 20, 2026 14:32:00
	mockNow := time.Date(2026, time.May, 20, 14, 32, 0, 0, msk)

	// Target weekday is Wednesday (today)
	// Case 1: target time is in the future (20:00) -> should be today (May 20)
	t1 := getNextWeekdayDate(mockNow, time.Wednesday, 20, 0)
	if t1.Day() != 20 {
		t.Errorf("getNextWeekdayDate(SameDay, FutureTime) = Day %d, want 20", t1.Day())
	}

	// Case 2: target time is in the past (10:00) -> should be next week (May 27)
	t2 := getNextWeekdayDate(mockNow, time.Wednesday, 10, 0)
	if t2.Day() != 27 {
		t.Errorf("getNextWeekdayDate(SameDay, PastTime) = Day %d, want 27", t2.Day())
	}

	// Case 3: target time has no time (hour = -1) -> should be today (May 20)
	t3 := getNextWeekdayDate(mockNow, time.Wednesday, -1, 0)
	if t3.Day() != 20 {
		t.Errorf("getNextWeekdayDate(SameDay, NoTime) = Day %d, want 20", t3.Day())
	}
}
