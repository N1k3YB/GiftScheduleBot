package parser

import (
	"regexp"
	"strings"
	"time"
	"unicode"
)

var giveawayKeywords = regexp.MustCompile(
	`(?i)(розыгрыш|разыгрываем|разыгрывается|разыгрывае|giveaway|раздаём|раздаем|раздача|розыгрышь|конкурс|призы\s+розыгрыш)`,
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
	reDMY     = regexp.MustCompile(`\b(\d{1,2})[./\-](\d{1,2})[./\-](\d{2,4})\b`)
	reDMonthY = regexp.MustCompile(`(?i)\b(\d{1,2})\s+(января|февраля|марта|апреля|мая|июня|июля|августа|сентября|октября|ноября|декабря|january|february|march|april|june|july|august|september|october|november|december)\s*(\d{4})?\b`)
	reDMonth  = regexp.MustCompile(`(?i)\b(\d{1,2})\s+(января|февраля|марта|апреля|мая|июня|июля|августа|сентября|октября|ноября|декабря)\b`)
)

type GiveawayInfo struct {
	IsGiveaway        bool
	Title             string
	Prizes            []string
	EndDate           *time.Time
	ResultsInSamePost bool
	HasResults        bool
}

func ParseGiveaway(text string) GiveawayInfo {
	info := GiveawayInfo{}
	if !giveawayKeywords.MatchString(text) {
		return info
	}
	info.IsGiveaway = true
	info.Title = extractTitle(text)
	info.Prizes = extractPrizes(text)
	info.EndDate = extractDate(text)
	info.ResultsInSamePost = samePostResultsKeywords.MatchString(text)
	info.HasResults = resultsKeywords.MatchString(text)
	return info
}

func IsGiveawayText(text string) bool {
	return giveawayKeywords.MatchString(text)
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

func extractDate(text string) *time.Time {
	now := time.Now()
	if m := reDMY.FindStringSubmatch(text); m != nil {
		d, mo, y := parseInt(m[1]), parseInt(m[2]), parseInt(m[3])
		if y < 100 {
			y += 2000
		}
		if d > 0 && d <= 31 && mo > 0 && mo <= 12 {
			t := time.Date(y, time.Month(mo), d, 23, 59, 0, 0, time.Local)
			if t.After(now) {
				return &t
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
			t := time.Date(y, mo, d, 23, 59, 0, 0, time.Local)
			if t.After(now) {
				return &t
			}
		}
	}
	if m := reDMonth.FindStringSubmatch(text); m != nil {
		d := parseInt(m[1])
		mo := monthNames[strings.ToLower(m[2])]
		y := now.Year()
		if d > 0 && d <= 31 && mo > 0 {
			t := time.Date(y, mo, d, 23, 59, 0, 0, time.Local)
			if t.Before(now) {
				t = time.Date(y+1, mo, d, 23, 59, 0, 0, time.Local)
			}
			return &t
		}
	}
	return nil
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
