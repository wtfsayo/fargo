package tui2

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vrypan/fargo/farcaster"
	"github.com/vrypan/fargo/fctools"
	"github.com/vrypan/fargo/tui"
)

type castsBlock struct {
	id     string
	text   string
	height int
}

type CastsModel struct {
	cursor    int
	casts     fctools.CastGroup
	width     int
	height    int
	blocks    []castsBlock
	viewStart int
	viewEnd   int
	hashIdx   []fctools.Hash
}

func NewCastsModel() *CastsModel {
	// test vals 3, "8cbf9d20658bc99b91e38ae77bc5c34cc53da972"
	return &CastsModel{}
}

func (m *CastsModel) LoadCasts(fid uint64, hash []byte) *CastsModel {
	casts := fctools.NewCastGroup().FromCast(nil, &farcaster.CastId{Fid: fid, Hash: hash}, true)
	m.casts = *casts
	m.blocks = make([]castsBlock, len(casts.Messages))
	m.hashIdx = make([]fctools.Hash, len(casts.Messages))
	m.cursor = 0
	m.viewStart = 0
	m.viewEnd = 0
	m.appendBlocks(nil, 0)
	return m
}
func (m *CastsModel) LoadFid(fid uint64) *CastsModel {
	casts := fctools.NewCastGroup().FromFid(nil, fid, 50)
	m.casts = *casts
	m.blocks = make([]castsBlock, len(casts.Messages))
	m.hashIdx = make([]fctools.Hash, len(casts.Messages))
	m.cursor = 0
	m.viewStart = 0
	m.viewEnd = 0
	m.appendBlocks(nil, 0)
	return m
}

func (m *CastsModel) Init() tea.Cmd {
	return nil
}

func (m *CastsModel) initViewport() {
	m.viewStart = 0
	height := 0
	for i, b := range m.blocks {
		if height+b.height+1 > m.height {
			m.cursor = 0
			break
		}
		m.viewEnd = i
		height += b.height + 1
	}
}

func (m *CastsModel) appendBlocks(hash *fctools.Hash, padding int) {
	opts := tui.FmtCastOpts{Width: 80}
	if hash == nil && m.casts.Head != (fctools.Hash{}) {
		hash = &m.casts.Head
	}
	if hash != nil { // This is a thread
		m.handleThreadBlocks(hash, padding, opts)
	} else {
		m.handleListBlocks(padding, opts)
	}
}

func (m *CastsModel) handleThreadBlocks(hash *fctools.Hash, padding int, opts tui.FmtCastOpts) {
	text := tui.FmtCast(m.casts.Messages[*hash].Message, m.casts.Fnames, padding, padding == 0, &opts)
	m.blocks[m.cursor] = castsBlock{
		id:     hash.String(),
		text:   text,
		height: strings.Count(text, "\n") + 1,
	}
	m.hashIdx[m.cursor] = *hash
	m.cursor++
	for _, reply := range m.casts.Messages[*hash].Replies {
		m.appendBlocks(&reply, padding+4)
	}
}

func (m *CastsModel) handleListBlocks(padding int, opts tui.FmtCastOpts) {
	for i, hash := range m.casts.Ordered {
		msg := m.casts.Messages[hash]
		text := tui.FmtCast(msg.Message, m.casts.Fnames, padding, padding == 0, &opts)
		m.hashIdx[i] = hash
		m.blocks[i] = castsBlock{
			id:     hash.String(),
			text:   text,
			height: strings.Count(text, "\n") + 1,
		}
	}
}

func (m *CastsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			m.moveCursorUp()
		case "down", "j":
			m.moveCursorDown()
			/*
				case "enter":
					hash := m.hashIdx[m.cursor]
					fid := m.casts.Messages[hash].Message.Data.Fid
					m.LoadCasts(fid, hash.Bytes())
			*/
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m *CastsModel) moveCursorUp() {
	if m.cursor > 0 {
		m.cursor--
		if m.cursor < m.viewStart {
			m.viewStart = m.cursor
			m.recalculateViewEnd()
		}
	}
}

func (m *CastsModel) moveCursorDown() {
	if m.viewEnd == 0 {
		m.initViewport()
	}
	if m.cursor < len(m.blocks)-1 {
		m.cursor++
		if m.cursor > m.viewEnd {
			m.viewEnd = m.cursor
			m.recalculateViewStart()
		}
	}
}

func (m *CastsModel) recalculateViewEnd() {
	height := 0
	for i := m.viewStart; i < len(m.blocks); i++ {
		if height+m.blocks[i].height+1 > m.height {
			m.viewEnd = i - 1
			return
		}
		height += m.blocks[i].height + 1
	}
	m.viewEnd = len(m.blocks)
}

func (m *CastsModel) recalculateViewStart() {
	height := 0
	i := m.viewEnd
	for ; i >= 0 && height+m.blocks[i].height+1 <= m.height; i-- {
		height += m.blocks[i].height + 1
	}
	m.viewStart = i + 1
}

func (m *CastsModel) View() string {
	var s strings.Builder
	if m.viewEnd == 0 {
		m.initViewport()
	}
	height := 0
	for i := m.viewStart; height < m.height && i < len(m.blocks); i++ {
		style := lipgloss.NewStyle().Bold(m.cursor == i)
		if height+m.blocks[i].height+1 < m.height {
			s.WriteString(fmt.Sprintf("%s\n", style.Render(m.blocks[i].text)))
			height += m.blocks[i].height
		} else {
			lines := strings.Split(m.blocks[i].text, "\n")
			for _, line := range lines {
				s.WriteString(fmt.Sprintf("%s\n", style.Render(line)))
				height++
				if height == m.height {
					break
				}
			}
		}
	}
	s.WriteString(strings.Repeat("\n", m.height-height))
	//s.WriteString(fmt.Sprintf("\nPress q to quit. %d %d-%d of %d. Height=%d/%d",
	//	m.cursor, m.viewStart, m.viewEnd, len(m.blocks), height, m.height))
	return s.String()
}

func (m *CastsModel) Status() (int, uint64, []byte) {
	hash := m.hashIdx[m.cursor]
	fid := m.casts.Messages[hash].Message.Data.Fid
	return m.cursor, fid, hash.Bytes()
}
