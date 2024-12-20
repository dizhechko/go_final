package scheduler

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/dizhechko/todo-list-server/settings"
)

// Структура правил повторения задачи
type RepeatRules struct {
	DatePart string  // часть даты d,m,y или w
	nums     [][]int // дополнительные параметры
}

// Слайс часть даты в правилах повторения задач
var PossibleVals = []string{"d", "m", "y", "w"}

// LastDayOfMonth определяет последнее число месяца
func LastDayOfMonth(date time.Time) int {
	y, m, _ := date.Date()
	ld := time.Date(y, m+1, 0, 0, 0, 0, 0, time.Local)
	return ld.Day()
}

// NextDate возвращает следующую дату повторения задачи в формате 20060102 и ошибку.
// Возвращаемая дата должна быть больше даты, указанной в переменной now.
//
//	now — время от которого ищется ближайшая дата
//	date — исходное время в формате 20060102, от которого начинается отсчёт повторений
//	repeat — правило повторения
func NextDate(now string, date string, repeat string) (string, error) {
	date = strings.TrimSpace(date)
	begDate, err := time.Parse(settings.DateFormat, date)
	if err != nil {
		return "", err
	}

	now = strings.TrimSpace(now)
	nowDate, err := time.Parse(settings.DateFormat, now)
	if err != nil {
		return "", err
	}

	rules, err := ParseRepeat(repeat)
	if err != nil {
		return "", err
	}
	if rules.DatePart == "" {
		return "", nil
	}

	var nextDate time.Time

	greaterDate := nowDate // выясняем какая дата больше now или дата начала отсчета
	if nowDate.Before(begDate) {
		greaterDate = begDate
	}

	switch rules.DatePart {
	case "y":
		diff := nowDate.Year() - begDate.Year()
		if diff > 0 {
			nextDate = begDate.AddDate(diff, 0, 0)
		} else {
			// Если дата начала задачи больше текущей даты, то нужно брать дату начала
			nextDate = begDate.AddDate(1, 0, 0)
		}

	case "m":
		return "", errors.New("Формат не поддерживается 'm'")

	case "d":
		if len(rules.nums) != 1 {
			return "", errors.New("Неправильное значение для даты 'd'")
		}
		if len(rules.nums[0]) != 1 {
			return "", errors.New("Неправильное значение для даты 'd'")
		}
		if rules.nums[0][0] > 400 {
			return "", errors.New("Неправильное значение для даты 'd', макс значение 400")
		}
		if nowDate.Before(begDate) {
			nextDate = begDate.AddDate(0, 0, rules.nums[0][0])
		} else {

			if begDate.Equal(nowDate) {
				nextDate = nowDate
			} else {
				daysCnt := int(nowDate.Sub(begDate).Abs().Hours())/24/rules.nums[0][0] + 1
				nextDate = begDate.AddDate(0, 0, rules.nums[0][0]*daysCnt)
			}
			//}
		}

	case "w":
		if len(rules.nums) != 1 {
			return "", errors.New("invalid number of additional arguments for date part 'w'")
		}
		if len(rules.nums[0]) < 1 {
			return "", errors.New("invalid number of additional arguments for date part 'w'")
		}
		// находим минимальную положительную разницу между текущим днем недели и днями из правила
		weekday := int(greaterDate.Weekday())
		minDiff := 8
		for _, wd := range rules.nums[0] {
			if wd < 1 || wd > 7 {
				return "", errors.New("invalid number of additional arguments for date part 'w', value must be between 1 and 7")
			}

			curDiff := 0
			if wd > weekday {
				curDiff = wd - weekday
			} else {
				curDiff = 7 - weekday + wd
			}

			if curDiff < minDiff {
				minDiff = curDiff
			}
		}
		nextDate = greaterDate.AddDate(0, 0, minDiff)

	}

	return nextDate.Format(settings.DateFormat), nil
}

// ParseRepeat парсит правило повторения задач repeat и возвращает результат в виде структуры RepeatRules
func ParseRepeat(repeat string) (RepeatRules, error) {
	if repeat := strings.TrimSpace(repeat); repeat == "" {
		return RepeatRules{DatePart: ""}, nil
	}

	repeatRules := RepeatRules{}
	// разделяем правило на слова и проверяем входит ли первая буква (слово) в список допустимых значений
	rules := strings.Split(repeat, " ")
	if !slices.Contains(PossibleVals, rules[0]) {
		return RepeatRules{}, errors.New("Отклонение от правил")
	}
	repeatRules.DatePart = rules[0]

	// парсим правило и попутно проверяем на ошибки формата
	for i, v := range rules[1:] {
		num, err := strconv.Atoi(v)
		if err == nil {
			repeatRules.nums = append(repeatRules.nums, []int{num})
		} else {
			for _, e := range strings.Split(v, ",") {
				num, err = strconv.Atoi(e)
				if err != nil {
					return RepeatRules{}, errors.New("Нарушен формат дней")
				}
				if len(repeatRules.nums) < i+1 {
					repeatRules.nums = append(repeatRules.nums, []int{})
				}
				repeatRules.nums[i] = append(repeatRules.nums[i], num)
			}
		}
	}

	return repeatRules, nil
}
