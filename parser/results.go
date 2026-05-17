package parser

import (
	"regexp"
	"strings"
	"time"
)

var resultsPostKeywords = regexp.MustCompile(
	`(?i)(итог(и)?|результат(ы)?|победитель|winners?|подводим итог|объявляем|поздравляем победителя)`,
)

type CheckResult int

const (
	ResultFound    CheckResult = iota
	ResultNotFound CheckResult = iota
	ResultUnknown  CheckResult = iota
)

type WinnerCheckResult struct {
	Status  CheckResult
	Message string
}

func CheckWinner(postText, username, firstName, lastName string, endDate *time.Time) WinnerCheckResult {
	if postText == "" {
		return WinnerCheckResult{
			Status:  ResultUnknown,
			Message: "❓ Текст поста с результатами недоступен. Проверь вручную.",
		}
	}

	if !resultsPostKeywords.MatchString(postText) {
		return WinnerCheckResult{
			Status:  ResultUnknown,
			Message: "❓ Похоже, итоги ещё не подведены или результаты не найдены в тексте.",
		}
	}

	lower := strings.ToLower(postText)

	if username != "" {
		uname := strings.ToLower(strings.TrimPrefix(username, "@"))
		if strings.Contains(lower, "@"+uname) || strings.Contains(lower, uname) {
			return WinnerCheckResult{
				Status:  ResultFound,
				Message: "🏆 Поздравляем! Кажется, ты в числе победителей! Проверь пост на всякий случай.",
			}
		}
	}

	if firstName != "" {
		fn := strings.ToLower(firstName)
		if strings.Contains(lower, fn) {
			if lastName != "" {
				ln := strings.ToLower(lastName)
				if strings.Contains(lower, ln) {
					return WinnerCheckResult{
						Status:  ResultFound,
						Message: "🏆 Поздравляем! Твоё имя найдено в результатах! Обязательно перепроверь в оригинале.",
					}
				}
			} else {
				return WinnerCheckResult{
					Status:  ResultFound,
					Message: "🏆 Имя найдено в результатах. Но убедись сам — совпадений может быть несколько.",
				}
			}
		}
	}

	return WinnerCheckResult{
		Status:  ResultNotFound,
		Message: "😔 Не нашли твои данные в результатах. Но лучше сам перепроверь — бот может ошибиться.",
	}
}

func IsResultsPost(text string) bool {
	return resultsPostKeywords.MatchString(text)
}

func ValidateResultsDate(resultsDate, endDate *time.Time) bool {
	if resultsDate == nil || endDate == nil {
		return true
	}
	return !resultsDate.Before(*endDate)
}

func ExtractResultsKeywords(text string) bool {
	return resultsPostKeywords.MatchString(text)
}
