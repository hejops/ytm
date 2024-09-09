package main

import (
	"os/exec"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	searchResults []YoutubeVideo
	searchTable   *table.Model

	// eventually we want 3 []/Model pairs: searchResults / albumSongs / queueSongs
	// searchTable and albumSongs will be rendered on top
	// queueSongs (if any) will be rendered on bottom

	searching bool
	input     string

	nowPlaying *YoutubeVideo
	ticker     *time.Ticker
}

var VideoColumns = []table.Column{
	{Title: "Title", Width: 20},
	{Title: "Artist", Width: 20},
	{Title: "Album", Width: 20},
	{Title: "Dur", Width: 5},
	{Title: "Plays", Width: 5},
}

func (v *YoutubeVideo) asRow() table.Row {
	// TODO: use reflect to access v fields via VideoColumns?
	return table.Row{v.Title, v.Artist, v.Album, v.Duration, v.Plays}
}

func (m *Model) Search() {
	results := parseCurlJq(searchCurlJq(m.input))
	m.searchResults = results

	var rows []table.Row
	for _, v := range results {
		rows = append(rows, v.asRow())
	}
	t := table.New(table.WithRows(rows), table.WithColumns(VideoColumns))
	m.searchTable = &t
}

// Init is the first function that will be called. It returns an optional
// initial command. To not perform an initial command return nil.
func (m *Model) Init() tea.Cmd {
	// TODO: set up mpv socket; playerctl is fine, but will almost
	// certainly mess with other running players

	m.searchResults = nil
	m.searching = true
	m.ticker = time.NewTicker(time.Second)
	return nil
}

// Update is called when a message is received. Use it to inspect messages
// and, in response, update the model and/or send a command.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
	case tea.KeyMsg:
		s := msg.String()

		if m.searching {
			switch s {
			case "enter":
				m.Search()
				m.searching = false
				m.input = ""
			case "backspace":
				if m.input == "" {
					m.searching = false
				} else {
					m.input = m.input[:len(m.input)-1]
				}
			default:
				m.input += string(msg.Runes)
			}
			return m, nil
		}

		switch s {

		case "q":
			return m, tea.Quit
		case "/":
			m.searching = true
		case "j":
			m.searchTable.MoveDown(1)
		case "k":
			m.searchTable.MoveUp(1)
		case "enter":
			// v := (*m.searchResults)[m.searchTable.Cursor()]
			v := m.searchResults[m.searchTable.Cursor()]
			url := "https://www.youtube.com/watch?v=" + v.Id

			return m, func() tea.Msg {
				m.nowPlaying = &v
				_ = exec.Command("mpv", "--force-window", url).Run()
				m.nowPlaying = nil
				// automatically redraw when mpv ends (a
				// regular go func won't do this)
				return tea.ClearScreen()
			}

		}
	}

	// TODO: if nowPlaying != nil, redraw every second

	return m, nil
}

// View renders the program's UI, which is just a string. The view is
// rendered after every Update.
func (m *Model) View() string {
	if len(m.searchResults) == 0 {
		switch {
		case m.searching && len(m.input) == 0:
			return "[Type to start searching]"
		case m.searching:
			return "/" + m.input
		default:
			return "Press / to search"
		}
	}

	t := lipgloss.NewStyle().MaxHeight(15).Render(m.searchTable.View())
	switch m.nowPlaying != nil {

	case true:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			t,
			"Now playing: "+m.nowPlaying.Title,
		)

	case false:
		return t

	}
	panic("unreachable")
}
