# sitnag

Short for sit-nag. A sibling of [granag](https://github.com/vusty/granag).

A Windows tray tool that counts the time spent at the computer and, after an
hour, asks for a five-minute warm-up. I know what the warm-up is; what I keep
doing is sitting for three hours straight.

## Use

    sitnag run              the reminder, with the tray icon
    sitnag autostart on     start at logon, through the Run key; flags after "on" go to run
    sitnag autostart off
    sitnag autostart status

`run` takes `-remind 60m -limit 90m -away 10m -break 5m -busy 5s -short 5m -long 30m`
and `-log FILE`. Small values make a quick end-to-end check:
`-remind 15s -limit 45s -break 10s -short 10s -long 30s`.

Build from WSL, run on Windows:

    GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui -s -w" -o sitnag.exe .

## How it decides

Time at the computer grows while the keyboard or mouse is touched
(`GetLastInputInfo`). Ten minutes with no input, a locked screen, or sleep
count as a break and start the count over: walking away for lunch is a break,
and the reminder should not greet you when you come back. Five minutes for the
bathroom and a glass of water is not, so the count goes on through it.

After an hour a window comes up in the middle of the screen, on top of
everything:

- **Начать разминку**: a five-minute countdown in the same window, a sound at
  the end, and the count starts over. The countdown runs only while the
  keyboard and mouse are left alone: any input stops it for five seconds, and
  the window turns amber and says the computer is in use. Pressing the button
  and carrying on working gets no warm-up for free.
- **Отложить на 5 мин** and **Отложить на 30 мин**: the window goes and comes
  back after that long.

Past ninety minutes the 30-minute button is gone, and a 30-minute snooze taken
earlier never ends later than the ninety-minute mark. Postponing is allowed;
postponing past ninety minutes only five at a time. From there the window turns
red and each reminder carries the next fact about what sitting does to the
body (`facts.go`, each with its study).

Walking away while the window is up counts as the break: the window goes and
the count restarts.

The window never takes the keyboard focus (`WS_EX_NOACTIVATE`), so a keystroke
meant for the editor cannot press one of its buttons. It has no close box and
ignores Alt+F4: it goes away through its buttons.

It paints itself, buttons included, on a dark gradient: blue while asking, red
past ninety minutes, green during the warm-up, amber while the warm-up is
paused. GDI+ does the shapes, GDI the text. To look at all four without waiting
for them:

    GOOS=windows go test -c -o win.test.exe ./internal/win
    SITNAG_RENDER=C:\temp\sitnag win.test.exe -test.run Render

The tray dot is green while counting, orange while the reminder is up, blue
during the warm-up. Its tooltip and the first menu line say how long you have
been sitting and when a snoozed reminder returns.

## Limits

Idle is input only. An hour on a call spent listening without touching the
mouse reads as having left the computer.
