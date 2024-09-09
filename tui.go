package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
	"github.com/charmbracelet/x/term"
)

type Model struct {
	searchResults []YoutubeVideo
	searchTable   *table.Model
	showAlbumInfo bool

	// eventually we want 3 []/Model pairs: searchResults / albumSongs / queueSongs
	// searchTable and albumSongs will be rendered on top
	// queueSongs (if any) will be rendered on bottom

	playlist []*YoutubeVideo

	searching bool
	input     string

	width  int
	height int

	player     Player
	nowPlaying *YoutubeVideo
	ticker     *time.Ticker
}

func (v *YoutubeVideo) asRow() table.Row {
	// TODO: use reflect to access v fields via VideoColumns?
	return table.Row{
		v.Title,
		v.Artist.Name,
		v.Album.Name,
		v.Duration,
		v.Plays,
	}
}

// Columns with negative width are to be resized dynamically.
var videoColumns = []table.Column{
	{Title: "Title", Width: -1},
	{Title: "Artist", Width: -1},
	{Title: "Album", Width: -1},
	{Title: "Dur", Width: 5},
	{Title: "Plays", Width: 5},
}

func resizeColumns(maxWidth int) []table.Column {
	// i considered 2 implementations:
	//
	// 1. always load a static map of column widths, which contains info
	// about variable columns embedded as int (-1)
	//
	// 2. make no assumptions about the incoming map, but check a separate
	// map of fixed (or variable) column widths
	//
	// 1 is easier, and makes sense since we only have one set of columns
	// (for now) that never change

	columns := videoColumns
	newColumns := make([]table.Column, len(columns))

	// determine how many columns are fixed (and their total width)
	var fixedWidth int
	var flexCols int
	for i, col := range columns {
		if col.Width < 0 {
			flexCols++
		} else {
			newColumns[i] = col
			fixedWidth += col.Width
		}
	}

	for i, col := range columns {
		if col.Width < 0 {
			// no idea where this extra 9 comes from; this must be
			// abstracted away from callers
			newWidth := (maxWidth - fixedWidth - 9) / flexCols
			// log.Printf("resizing %s: %d -> %d (max w: %d)", col.Title, col.Width, newWidth, maxWidth)
			newColumns[i] = table.Column{Title: col.Title, Width: newWidth}
		}
	}

	return newColumns
}

// func resizeColumns(columns *[]table.Column, maxWidth int) {
// 	// newColumns := make([]table.Column, len(columns))
//
// 	// determine how many columns are fixed (and their total width)
// 	var fixedWidth int
// 	var flexCols int
// 	for _, col := range *columns {
// 		if col.Width < 0 {
// 			flexCols++
// 		} else {
// 			// newColumns[i] = col
// 			fixedWidth += col.Width
// 		}
// 	}
//
// 	// no idea where this extra 9 comes from; this must be abstracted away
// 	// from callers
// 	newWidth := (maxWidth - fixedWidth - 9) / flexCols
//
// 	for i, col := range *columns {
// 		if col.Width < 0 {
// 			// log.Printf("resizing %s: %d -> %d (max w: %d)", col.Title, col.Width, newWidth, maxWidth)
// 			(*columns)[i] = table.Column{Title: col.Title, Width: newWidth}
// 		}
// 	}
// }

func (m *Model) updateSearchTable() {
	var rows []table.Row
	for _, v := range m.searchResults {
		rows = append(rows, v.asRow())
	}

	t := table.New()

	// cols := videoColumns
	// resizeColumns(&cols, m.width-2)
	// t.SetColumns(cols)

	// columns must be set before rows (else panic)
	t.SetColumns(resizeColumns(m.width - 2))
	t.SetRows(rows)

	// dims and contents can be set in any order
	t.SetHeight(m.height/2 - 1)
	t.SetWidth(m.width)

	m.searchTable = &t
}

// Init is the first function that will be called. It returns an optional
// initial command. To not perform an initial command return nil.
func (m *Model) Init() tea.Cmd {
	m.player.init()
	m.searching = true
	m.ticker = time.NewTicker(time.Second)

	// go func() {
	// 	for {
	// 		<-m.ticker.C
	// 		// update progress
	// 	}
	// }()

	w, h, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		panic(err)
	}
	m.width = w
	m.height = h

	return nil
}

// Update is called when a message is received. Use it to inspect messages
// and, in response, update the model and/or send a command.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:

		m.height = msg.Height
		m.width = msg.Width

		if m.searchTable != nil {
			m.updateSearchTable()
		}

		return m, tea.ClearScreen

	case tea.KeyMsg:
		s := msg.String()

		if m.searching {
			switch s {
			case "enter":
				m.searchResults = parseCurlJq(searchCurlJq(m.input))
				m.searching = false
				m.updateSearchTable()
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
			m.player.quit()
			return m, tea.Quit
		case "/":
			m.searching = true
			m.input = ""

		// the bottom half is -not- meant to be navigable (for now)

		case "j":
			m.searchTable.MoveDown(1)
		case "k":
			m.searchTable.MoveUp(1)

		case " ":
			// TODO: toggle (cycle pause) does nothing if playback stopped
			m.player.toggle()

		case "i": // toggle album info (upper pane, right half)
			m.showAlbumInfo = !m.showAlbumInfo

		case ">":
			m.player.next()
		case "<":
			m.player.prev()

		case "a":
			v := m.searchResults[m.searchTable.Cursor()]
			m.playlist = append(m.playlist, &v)

			url := "https://www.youtube.com/watch?v=" + v.Id
			m.player.enqueue(url)
			return m, nil

		case "enter":
			v := m.searchResults[m.searchTable.Cursor()]
			m.nowPlaying = &v
			m.playlist = nil
			m.playlist = append(m.playlist, &v)

			url := "https://www.youtube.com/watch?v=" + v.Id
			m.player.play(url)
			return m, nil

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

	var panes []string

	// 1 header
	header := lipgloss.NewStyle().
		// BorderStyle(lipgloss.NormalBorder()).
		MaxHeight(3).
		Render(" Search results: " + m.input)

	panes = append(panes, header)

	// 2 search/album
	switch m.showAlbumInfo {

	case true:
		cursor := m.searchTable.Cursor()
		v := m.searchResults[cursor]

		panes = append(panes, strings.Join([]string{
			v.Title,
			v.Artist.Name,
			v.Album.Name,
			// "https://music.youtube.com/browse/" + v.Album.Id,
			"https://music.youtube.com/playlist?list=" + v.Album.getPlaylistId(),
		}, "\n"))

	case false:
		// TODO: style currently playing row (seems non-trivial)
		t := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false).
			MaxWidth(m.width - 2).
			MaxHeight(m.height/2 + 1).
			Render(m.searchTable.View())

		panes = append(panes, t)
	}

	// 3 playlist
	if len(m.playlist) > 0 {
		playlist := list.New()

		for _, q := range m.playlist {
			playlist.Item(q.Title)
		}

		panes = append(
			panes,
			lipgloss.NewStyle().MaxHeight(m.height/2-1).Render(playlist.String()),
		)
	}

	// 4 now playing
	if m.nowPlaying != nil {
		panes = append(
			panes,
			// TODO: progress
			fmt.Sprintf("Now playing: %s", m.nowPlaying.Title),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, panes...)
}
