//go:build windows

package main

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/vusty/sitnag/internal/figure"
)

// texts is every string the tool shows, in one language. Formats take their
// arguments in the order the comments give.
type texts struct {
	lang string

	start, snooze string // snooze: duration

	askTitle, askCaption       string // askCaption: break length
	urgentTitle, urgentCaption string // urgentCaption: time sitting, short snooze
	factHeading                string

	breakTitle, breakNote   string
	pausedTitle, pausedNote string
	moves                   [figure.Count]moveText // in the order of figure.Move

	trayAsk, trayBreak, trayPaused string // time sitting; time left; time left
	trayIdle, traySitting          string // -; time sitting
	trayReturns                    string // clock time
	trayStarting, quit             string
	breakNow                       string

	sec, min, hour, hourMin string // number formats for durations
}

// moveText is what the warm-up says next to the mannequin: what the move is
// called, and how to do it in a line or two.
type moveText struct{ name, cue string }

var russian = texts{
	lang:          "ru",
	start:         "Начать разминку",
	snooze:        "Отложить на %s",
	askTitle:      "Пора размяться",
	askCaption:    "за компом без перерыва. Встань, пройдись, потянись: %s, и снова за дело.",
	urgentTitle:   "Хватит сидеть",
	urgentCaption: "За компом уже %s без перерыва. Отложить можно только на %s.",
	factHeading:   "ПОКА ТЫ СИДИШЬ",
	breakTitle:    "Разминка",
	breakNote:     "Отсчёт идёт, пока клавиатура и мышь не трогаются.",
	pausedTitle:   "Разминка на паузе",
	pausedNote:    "Ты пользуешься компьютером, отсчёт стоит. Отойди, и он пойдёт дальше.",
	moves: [figure.Count]moveText{
		figure.Neck:        {"Наклоны головы", "Ухо к плечу, плечи не поднимай. Медленно, в обе стороны по очереди."},
		figure.Shoulders:   {"Круги плечами", "Вперёд, вверх, назад, вниз. Руки расслаблены, лопатки своди."},
		figure.ArmCircles:  {"Круги руками", "Прямые руки, большой круг через верх. Потом в обратную сторону."},
		figure.Reach:       {"Потянись вверх", "Руки через стороны вверх, поднимись на носки и тянись за ладонями."},
		figure.SideBend:    {"Наклоны в стороны", "Одна рука над головой, тянись за ней вбок. Таз остаётся на месте."},
		figure.Twist:       {"Повороты корпуса", "Руки в стороны, поворачивай плечи и грудь. Стопы стоят на месте."},
		figure.ForwardBend: {"Наклон вперёд", "Спина расслаблена, руки висят к полу. Задержись внизу, поднимайся плавно."},
		figure.Squat:       {"Приседания", "Таз назад, как на стул, руки вперёд. Пятки от пола не отрывай."},
		figure.CalfRaise:   {"Подъёмы на носки", "Поднимись на носки, задержись на секунду и медленно опустись."},
		figure.March:       {"Ходьба на месте", "Колени высоко, руки работают в такт. В бодром темпе."},
	},
	trayAsk:      "пора размяться, за компом %s",
	trayBreak:    "разминка, осталось %s",
	trayPaused:   "разминка на паузе: ты за компом, осталось %s",
	trayIdle:     "за компом никого, счёт с нуля",
	traySitting:  "за компом %s",
	trayReturns:  ", напомню в %s",
	trayStarting: "считаю время за компом",
	quit:         "Выход",
	breakNow:     "Начать разминку сейчас",
	sec:          "%d сек",
	min:          "%d мин",
	hour:         "%d ч",
	hourMin:      "%d ч %02d мин",
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
	breakTitle:    "Warm-up",
	breakNote:     "The clock runs while the keyboard and mouse are left alone.",
	pausedTitle:   "Warm-up paused",
	pausedNote:    "You are using the computer, so the clock stands still. Step away and it goes on.",
	moves: [figure.Count]moveText{
		figure.Neck:        {"Neck tilts", "Ear towards the shoulder, shoulders stay down. Slowly, one side then the other."},
		figure.Shoulders:   {"Shoulder circles", "Forward, up, back, down. Arms loose, squeeze the shoulder blades."},
		figure.ArmCircles:  {"Arm circles", "Straight arms, one big circle over the top. Then the other way round."},
		figure.Reach:       {"Reach up", "Arms out and up, rise onto your toes and reach after your hands."},
		figure.SideBend:    {"Side bends", "One arm over the head, reach sideways after it. Hips stay where they are."},
		figure.Twist:       {"Torso twists", "Arms out, turn the shoulders and chest. Feet stay planted."},
		figure.ForwardBend: {"Forward bend", "Back loose, arms hanging to the floor. Hold at the bottom, come up slowly."},
		figure.Squat:       {"Squats", "Hips back as if onto a chair, arms forward. Heels stay on the floor."},
		figure.CalfRaise:   {"Calf raises", "Up onto your toes, hold for a second, come down slowly."},
		figure.March:       {"March in place", "Knees high, arms swinging in time. Keep it brisk."},
	},
	trayAsk:      "time to move, %s at the computer",
	trayBreak:    "warm-up, %s left",
	trayPaused:   "warm-up paused: you are at the computer, %s left",
	trayIdle:     "nobody at the computer, count from zero",
	traySitting:  "%s at the computer",
	trayReturns:  ", back at %s",
	trayStarting: "counting time at the computer",
	quit:         "Quit",
	breakNow:     "Start the warm-up now",
	sec:          "%d s",
	min:          "%d min",
	hour:         "%d h",
	hourMin:      "%d h %02d min",
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
