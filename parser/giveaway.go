package parser

import (
	"regexp"
	"strings"
	"time"
	"unicode"
)

var msk = time.FixedZone("MSK", 3*3600)

func NormalizeText(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 0xE000 && r <= 0xF8FF:
			continue
		case r >= 0xFE00 && r <= 0xFE0F:
			continue
		case r >= 0x2000 && r <= 0x200F:
			continue
		case r >= 0x2028 && r <= 0x202F:
			continue
		case r == 0x00AD:
			continue
		case r == 0x034F:
			continue
		case r == 0x180E:
			continue
		case r == 0x2060:
			continue
		case r == 0xFEFF:
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

var giveawayKeywords = regexp.MustCompile(
	`(?i)(розыгрыш|разыгрываем|разыгрывается|разыгрывае|giveaway|раздаём|раздаем|раздача|розыгрышь|конкурс|призы\s+розыгрыш|итоги\s+розыгрыша|розыгрыша)`,
)

var prizeKeywords = regexp.MustCompile(
	`(?i)(приз|получит|выиграет|победитель|победители|награда|выигрыш|подарок|вручим|вручаем)`,
)

var resultsKeywords = regexp.MustCompile(
	`(?i)(итог|результат|подведём итог|подводим итог|winners?|объявляем победителя)`,
)

var samePostResultsKeywords = regexp.MustCompile(
	`(?i)(итоги\s+(в\s+)?(этом|данном)\s+посте?|результаты\s+(будут\s+)?(здесь|тут|в\s+этом)|обновим\s+пост|дополним\s+пост)`,
)

var monthNames = map[string]time.Month{
	"января": time.January, "февраля": time.February, "марта": time.March,
	"апреля": time.April, "мая": time.May, "июня": time.June,
	"июля": time.July, "августа": time.August, "сентября": time.September,
	"октября": time.October, "ноября": time.November, "декабря": time.December,
	"january": time.January, "february": time.February, "march": time.March,
	"april": time.April, "may": time.May, "june": time.June,
	"july": time.July, "august": time.August, "september": time.September,
	"october": time.October, "november": time.November, "december": time.December,
}

var (
	reDMY      = regexp.MustCompile(`\b(\d{1,2})[./\-](\d{1,2})[./\-](\d{2,4})\b`)
	reDMonthY  = regexp.MustCompile(`(?i)\b(\d{1,2})\s+(января|февраля|марта|апреля|мая|июня|июля|августа|сентября|октября|ноября|декабря|january|february|march|april|june|july|august|september|october|november|december)\s*(\d{4})?\b`)
	reDMonth   = regexp.MustCompile(`(?i)\b(\d{1,2})\s+(января|февраля|марта|апреля|мая|июня|июля|августа|сентября|октября|ноября|декабря)\b`)
	reTime     = regexp.MustCompile(`\b(?:в\s*)?(\d{1,2})[:. ](\d{2})\s*(?:мск|msk|по\s+москве)?\b`)
	reRelative = regexp.MustCompile(`(?i)\b(сегодня|завтра|послезавтра|через\s+(\d+)\s+д(?:ень|ня|ней))\b`)
)

type GiveawayInfo struct {
	IsGiveaway        bool
	Title             string
	Prizes            []string
	EndDate           *time.Time
	HasEndTime        bool
	ResultsInSamePost bool
	HasResults        bool
}

func ParseGiveaway(rawText string) GiveawayInfo {
	text := NormalizeText(rawText)
	info := GiveawayInfo{}
	if !giveawayKeywords.MatchString(text) {
		return info
	}
	info.IsGiveaway = true
	info.Title = extractTitle(text)
	info.Prizes = extractPrizes(text)
	info.EndDate, info.HasEndTime = extractDate(text)
	info.ResultsInSamePost = samePostResultsKeywords.MatchString(text)
	info.HasResults = resultsKeywords.MatchString(text)
	return info
}

func IsGiveawayText(text string) bool {
	return giveawayKeywords.MatchString(NormalizeText(text))
}

func extractTitle(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimFunc(line, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})
		if len(line) > 3 {
			if len(line) > 100 {
				return line[:100] + "…"
			}
			return line
		}
	}
	return "Розыгрыш"
}

func extractPrizes(text string) []string {
	var prizes []string
	lines := strings.Split(text, "\n")
	inPrizeSection := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if prizeKeywords.MatchString(trimmed) {
			inPrizeSection = true
		}
		if inPrizeSection && trimmed != "" && len(trimmed) > 2 {
			prize := strings.TrimLeft(trimmed, "•-–—*✅🎁🎀🏆🥇🥈🥉1234567890. ")
			prize = strings.TrimSpace(prize)
			if prize != "" && len(prize) > 2 {
				prizes = append(prizes, prize)
			}
		}
		if inPrizeSection && trimmed == "" {
			inPrizeSection = false
		}
		if len(prizes) >= 10 {
			break
		}
	}
	return prizes
}

func extractDate(text string) (*time.Time, bool) {
	now := time.Now().In(msk)

	if m := reRelative.FindStringSubmatch(text); m != nil {
		kw := strings.ToLower(m[1])
		var base time.Time
		switch {
		case strings.HasPrefix(kw, "сегодня"):
			base = now
		case strings.HasPrefix(kw, "завтра"):
			base = now.AddDate(0, 0, 1)
		case strings.HasPrefix(kw, "послезавтра"):
			base = now.AddDate(0, 0, 2)
		default:
			days := parseInt(m[2])
			if days > 0 {
				base = now.AddDate(0, 0, days)
			}
		}
		if !base.IsZero() {
			h, mi := extractTime(text)
			hasTime := h >= 0
			if !hasTime {
				h, mi = 0, 0
			}
			t := time.Date(base.Year(), base.Month(), base.Day(), h, mi, 0, 0, msk)
			return &t, hasTime
		}
	}

	if m := reDMY.FindStringSubmatch(text); m != nil {
		d, mo, y := parseInt(m[1]), parseInt(m[2]), parseInt(m[3])
		if y < 100 {
			y += 2000
		}
		if d > 0 && d <= 31 && mo > 0 && mo <= 12 {
			h, mi := extractTime(text)
			hasTime := h >= 0
			if !hasTime {
				h, mi = 0, 0
			}
			t := time.Date(y, time.Month(mo), d, h, mi, 0, 0, msk)
			if t.After(now) {
				return &t, hasTime
			}
		}
	}
	if m := reDMonthY.FindStringSubmatch(text); m != nil {
		d := parseInt(m[1])
		mo := monthNames[strings.ToLower(m[2])]
		y := now.Year()
		if m[3] != "" {
			y = parseInt(m[3])
		}
		if d > 0 && d <= 31 && mo > 0 {
			h, mi := extractTime(text)
			hasTime := h >= 0
			if !hasTime {
				h, mi = 0, 0
			}
			t := time.Date(y, mo, d, h, mi, 0, 0, msk)
			if t.After(now) {
				return &t, hasTime
			}
		}
	}
	if m := reDMonth.FindStringSubmatch(text); m != nil {
		d := parseInt(m[1])
		mo := monthNames[strings.ToLower(m[2])]
		y := now.Year()
		if d > 0 && d <= 31 && mo > 0 {
			h, mi := extractTime(text)
			hasTime := h >= 0
			if !hasTime {
				h, mi = 0, 0
			}
			t := time.Date(y, mo, d, h, mi, 0, 0, msk)
			if t.Before(now) {
				t = time.Date(y+1, mo, d, h, mi, 0, 0, msk)
			}
			return &t, hasTime
		}
	}
	return nil, false
}

func extractTime(text string) (hour, min int) {
	if m := reTime.FindStringSubmatch(text); m != nil {
		h, mm := parseInt(m[1]), parseInt(m[2])
		if h >= 0 && h <= 23 && mm >= 0 && mm <= 59 {
			return h, mm
		}
	}
	return -1, 0
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
