package main

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// https://github.com/minamotorin/youtube-fzf/blob/sub/youtube-fzf#L497
// https://github.com/raitonoberu/ytmusic/blob/0e5780514b1d0c9cfb2dd7b51b31f70f15460f47/request.go#L11

func main() {
	if _, err := os.Stat(JQ_SCRIPT); err != nil {
		log.Fatalln("jq script not found:", JQ_SCRIPT)
	}

	_, err := tea.NewProgram(&Model{}, tea.WithAltScreen()).Run()
	if err != nil {
		panic(err)
	}
}
