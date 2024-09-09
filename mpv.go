package main

import (
	"encoding/json"
	"os/exec"
	"strings"
)

// must resist the urge to use https://github.com/blang/mpv ...

type Player struct {
	// socket string
}

const MPV_SOCKET = "/tmp/mpv_ytm"

func (p *Player) init() {
	// playerctl is fine, but will almost certainly mess with other running
	// players

	cmd := exec.Command(
		"mpv",
		"--idle",
		"--video=no",
		"--no-config",
		"--input-ipc-server="+MPV_SOCKET,
	)

	go func() {
		_ = cmd.Run() // mpv runs indefinitely, until main process ended
		// log.Println("ended mpv")
	}()
}

func (p *Player) runCommand(args []string) {
	b, err := json.Marshal(
		map[string][]string{"command": args},
	)
	if err != nil {
		panic(err)
	}

	// log.Println("cmd: echo '", string(b), "' | socat "+MPV_SOCKET+" -")

	// https://github.com/held-m/mpv-web-gui/blob/07af2537ffb80529d0010cfb922aadad473e2d47/api/infr/mpv/client.go#L12

	cmd := exec.Command("echo", string(b))
	pipe, _ := cmd.StdoutPipe()
	defer pipe.Close()

	socat := exec.Command("socat", "-", MPV_SOCKET)
	socat.Stdin = pipe
	err = cmd.Start()
	if err != nil {
		panic(err)
	}
	_ = socat.Run()
}

// https://github.com/mpv-player/mpv/blob/master/DOCS/man/input.rst

// Replace append url to playlist
func (p *Player) enqueue(url string) { p.runCommand([]string{"loadfile", url, "append"}) }

// Replace entire playlist with current url
func (p *Player) play(url string) { p.runCommand([]string{"loadfile", url}) }

func (p *Player) next()   { p.runCommand([]string{"playlist-next"}) }
func (p *Player) prev()   { p.runCommand([]string{"playlist-prev"}) }
func (p *Player) quit()   { p.runCommand([]string{"quit"}) }
func (p *Player) toggle() { p.runCommand(strings.Fields("cycle pause")) }
