//go:build windows

package main

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// texts is every string the tool shows, in one language. Formats take their
// arguments in the order the comments give.
type texts struct {
	lang string

	start, snooze string // snooze: duration

	askTitle, askCaption       string // askCaption: break length
	urgentTitle, urgentCaption string // urgentCaption: time sitting, short snooze
	factHeading                string
	facts                      []string

	breakTitle, breakNote   string
	pausedTitle, pausedNote string

	trayAsk, trayBreak, trayPaused string // time sitting; time left; time left
	trayIdle, traySitting          string // -; time sitting
	trayReturns                    string // clock time
	trayStarting, quit             string

	sec, min, hour, hourMin string // number formats for durations
}

var russian = texts{
	lang:          "ru",
	start:         "Начать разминку",
	snooze:        "Отложить на %s",
	askTitle:      "Пора размяться",
	askCaption:    "за компом без перерыва. Встань, пройдись, потянись: %s, и снова за дело.",
	urgentTitle:   "Хватит сидеть",
	urgentCaption: "За компом уже %s без перерыва. Отложить можно только на %s.",
	factHeading:   "ПОКА ТЫ СИДИШЬ",
	facts:         factsRU,
	breakTitle:    "Разминка",
	breakNote:     "Отсчёт идёт, пока клавиатура и мышь не трогаются.",
	pausedTitle:   "Разминка на паузе",
	pausedNote:    "Ты пользуешься компьютером, отсчёт стоит. Отойди, и он пойдёт дальше.",
	trayAsk:       "пора размяться, за компом %s",
	trayBreak:     "разминка, осталось %s",
	trayPaused:    "разминка на паузе: ты за компом, осталось %s",
	trayIdle:      "за компом никого, счёт с нуля",
	traySitting:   "за компом %s",
	trayReturns:   ", напомню в %s",
	trayStarting:  "считаю время за компом",
	quit:          "Выход",
	sec:           "%d сек",
	min:           "%d мин",
	hour:          "%d ч",
	hourMin:       "%d ч %02d мин",
}

var english = texts{
	lang:          "en",
	start:         "Start the warm-up",
	snooze:        "Snooze %s",
	askTitle:      "Time to move",
	askCaption:    "at the computer without a break. Stand up, walk, stretch: %s, then back to it.",
	urgentTitle:   "Enough sitting",
	urgentCaption: "%s at the computer without a break. Only a %s snooze is left.",
	factHeading:   "WHILE YOU SIT",
	facts:         factsEN,
	breakTitle:    "Warm-up",
	breakNote:     "The clock runs while the keyboard and mouse are left alone.",
	pausedTitle:   "Warm-up paused",
	pausedNote:    "You are using the computer, so the clock stands still. Step away and it goes on.",
	trayAsk:       "time to move, %s at the computer",
	trayBreak:     "warm-up, %s left",
	trayPaused:    "warm-up paused: you are at the computer, %s left",
	trayIdle:      "nobody at the computer, count from zero",
	traySitting:   "%s at the computer",
	trayReturns:   ", back at %s",
	trayStarting:  "counting time at the computer",
	quit:          "Quit",
	sec:           "%d s",
	min:           "%d min",
	hour:          "%d h",
	hourMin:       "%d h %02d min",
}

// t is the language in use, set once at startup.
var t = &english

// pickLanguage resolves -lang. "auto" takes the first language in the user's
// language list (Settings > Time & language), not the Windows display
// language: plenty of Russian speakers run an English Windows with Russian
// first in that list, and that list is the one they chose themselves.
func pickLanguage(flag string) (*texts, error) {
	switch flag {
	case "ru":
		return &russian, nil
	case "en":
		return &english, nil
	case "auto":
	default:
		return nil, fmt.Errorf("-lang: want auto, ru or en, got %q", flag)
	}
	if first := firstUserLanguage(); first != "" {
		if isRussian(first) {
			return &russian, nil
		}
		return &english, nil
	}
	if lang, err := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME); err == nil && len(lang) > 0 && isRussian(lang[0]) {
		return &russian, nil
	}
	return &english, nil
}

func firstUserLanguage() string {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\International\User Profile`, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	langs, _, err := k.GetStringsValue("Languages")
	if err != nil || len(langs) == 0 {
		return ""
	}
	return langs[0]
}

func isRussian(tag string) bool {
	return tag == "ru" || strings.HasPrefix(strings.ToLower(tag), "ru-")
}
