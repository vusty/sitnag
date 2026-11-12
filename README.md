# sitnag

Short for sit-nag. A Windows tray tool that counts the time you spend at the
computer and, after an hour, asks for a five-minute warm-up. I know what the
warm-up is; what I kept doing was sitting for three hours straight.

![The reminder after an hour](docs/ask.png)

It is small, stays in the tray, and is hard to wave away: the window sits on
top of everything, has no close box, and a warm-up only counts if you actually
leave the keyboard.

## What it does

- **Counts real time at the computer.** The count grows while the keyboard or
  mouse is in use. Ten minutes without input, a locked screen or sleep count as
  a break and start it over; five minutes for coffee do not.
- **Asks after an hour.** A window in the middle of the screen offers to start
  the warm-up or to snooze for 5 or 30 minutes.
- **Stops bargaining at ninety minutes.** From there only the 5-minute snooze
  is left, a 30-minute one taken earlier ends at the ninety-minute mark, and
  every reminder turns red and carries a fact about what sitting does to the
  body, with the study it comes from. The whole list is in
  [facts.go](facts.go).
- **Shows what to do.** A mannequin goes through ten moves, half a minute
  each, from the neck down to marching in place, with a line on how to do
  each one.
- **Counts only a real warm-up.** The five-minute countdown runs only while
  the keyboard and mouse are left alone. Use them for three seconds and it
  stops, and the seconds it ran meanwhile are taken back, so pressing "start"
  and carrying on working earns nothing. A stray touch on the way out of the
  chair does not count.
- **Lets you put it off.** A warm-up started by a slip of the hand, or one
  that something urgent cuts short, has a snooze button of its own. Five
  minutes later the reminder is back, and the warm-up starts from the top.
- **Lets you go early.** "Start the warm-up now" in the tray menu opens the
  countdown at any moment: tired before the hour, or ready before a 30-minute
  snooze runs out. Once it is done the count starts over, as after any warm-up.

| Past ninety minutes | The warm-up | Paused while you use the computer |
|---|---|---|
| ![Urgent reminder](docs/urgent.png) | ![Warm-up](docs/warmup.png) | ![Paused warm-up](docs/paused.png) |

The interface is in English or Russian, picked from the first language in the
Windows language list (Settings > Time & language). `-lang en` or `-lang ru`
overrides it.

## Install

Build it from source, or download `sitnag.exe` from
[Releases](https://github.com/vusty/sitnag/releases). The exe is not signed,
so building it yourself is the better option. Every step below is a command,
so a coding agent with shell access can do it too: "install sitnag from
github.com/vusty/sitnag following its README".

### Build from source

You need Windows 10 or 11 (x64), [Go](https://go.dev/dl/) 1.27 or newer, and
git. In PowerShell:

```powershell
# Go and git, if they are not installed yet; open a new terminal afterwards
winget install GoLang.Go Git.Git

git clone https://github.com/vusty/sitnag.git
cd sitnag
go build -ldflags "-H=windowsgui -s -w" -o sitnag.exe .

$dir = "$env:LOCALAPPDATA\Programs\sitnag"
New-Item -ItemType Directory -Force $dir | Out-Null
Copy-Item sitnag.exe $dir
```

From Linux, macOS or WSL the same binary cross-compiles, no cgo involved:

```sh
GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui -s -w" -o sitnag.exe .
```

### Or download it

```powershell
$dir = "$env:LOCALAPPDATA\Programs\sitnag"
New-Item -ItemType Directory -Force $dir | Out-Null
Invoke-WebRequest https://github.com/vusty/sitnag/releases/latest/download/sitnag.exe -OutFile "$dir\sitnag.exe"
Unblock-File "$dir\sitnag.exe"
```

### Start it with Windows

```powershell
$dir = "$env:LOCALAPPDATA\Programs\sitnag"
$log = "$env:LOCALAPPDATA\sitnag\sitnag.log"
& "$dir\sitnag.exe" autostart on -log $log | Out-Host
Start-Process "$dir\sitnag.exe" -ArgumentList "run", "-log", $log
```

A green dot appears in the tray. Hover over it to see how long you have been
sitting. The log is the only way to see afterwards what it decided and when.

### Uninstall

```powershell
& "$env:LOCALAPPDATA\Programs\sitnag\sitnag.exe" autostart off | Out-Host
```

Then choose Quit in the tray menu and delete `%LOCALAPPDATA%\Programs\sitnag`
and the log folder `%LOCALAPPDATA%\sitnag`.

## Settings

All settings are flags of `run`. Anything after `autostart on` is stored and
passed to `run` at logon, so to change a setting run `autostart on` again with
the new flags, then restart sitnag from the tray.

| Flag | Default | What it sets |
|---|---|---|
| `-remind` | `60m` | time at the computer before the first reminder |
| `-limit` | `90m` | past this only the short snooze is left, and the reminder turns red |
| `-away` | `10m` | no input this long counts as having left, and the count starts over |
| `-break` | `5m` | length of the warm-up |
| `-grace` | `3s` | input that goes on this long pauses the warm-up |
| `-busy` | `5s` | a paused warm-up goes on after this long without input |
| `-short`, `-long` | `5m`, `30m` | the two snoozes; the short one also puts off a running warm-up |
| `-lang` | `auto` | `auto`, `en` or `ru` |
| `-log` | none | append a log of decisions to this file |

For example, a reminder every 45 minutes:

```powershell
& "$env:LOCALAPPDATA\Programs\sitnag\sitnag.exe" autostart on -remind 45m -limit 75m -log "$env:LOCALAPPDATA\sitnag\sitnag.log" | Out-Host
```

Small values make a quick end-to-end check:
`sitnag.exe run -remind 15s -limit 45s -break 10s -short 10s -long 30s`.

## How it decides

The time comes from `GetLastInputInfo`, the moment of the last keyboard or
mouse input anywhere in the session. The tool checks it once a second.

- Ten minutes with no input, a locked screen, or a gap that means the machine
  slept: you left, and the count starts over. Walking away while the reminder
  is up counts as the break too, and the window goes.
- During the warm-up the clock runs unless the computer is in use. Use is
  input that goes on for three seconds; a run of input ends after three
  seconds without any. Then the window turns amber and says so, the clock
  goes back to where it was when that input began, and it starts again after
  five seconds without input. The click on the start button and a single
  touch after it do not count as use.
- A warm-up put off is still owed: the reminder comes back when the snooze
  ends, even if the hour is not up, and keeps coming back until the warm-up
  is done or you have been away for ten minutes.
- The move on screen follows the countdown: a new one every thirty seconds of
  warm-up actually done, so a pause holds the move it caught.
- The window never takes the keyboard focus (`WS_EX_NOACTIVATE`), so a
  keystroke meant for the editor cannot press one of its buttons. It ignores
  Alt+F4: it goes away through its buttons, or through Quit in the tray.

The tray dot is green while counting, orange while the reminder is up, and
blue during the warm-up.

## Limits

Idle is input only. An hour on a call spent listening without touching the
mouse reads as having left the computer, and reading a long page without
scrolling during the warm-up reads as being away.

## Development

    go test ./internal/sit          the decision logic; runs on any OS
    GOOS=windows go vet ./...

The window paints itself on a gradient: GDI+ draws the shapes, GDI the text.
To look at all four screens without waiting for them, build the render test and
run it on Windows; it writes the PNGs this README shows:

    GOOS=windows go test -c -o win.test.exe ./internal/win
    SITNAG_RENDER=C:\temp\sitnag win.test.exe -test.run Render

[granag](https://github.com/vusty/granag) is its sibling: it reminds you to
start recording in Granola when you unmute the microphone.

## License

MIT, see [LICENSE](LICENSE).
