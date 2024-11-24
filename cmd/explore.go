package cmd

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/vrypan/fargo/config"
	db "github.com/vrypan/fargo/localdb"
	"github.com/vrypan/fargo/tui2"
	"github.com/vrypan/fargo/tui2/history"
)

var exploreCmd = &cobra.Command{
	Use:   "explore [@username]/casts",
	Short: "Interactive Farcaster explorer",
	Long: `It only supports "@username/casts" for now.
Ex.: fargo explore @dwr/casts
`,
	Run: exploreRun,
}

func exploreRun(cmd *cobra.Command, args []string) {
	countFlag := uint32(config.GetInt("get.count"))

	db.Open()
	defer db.Close()
	user, parts := parse_url(args)
	if user == nil {
		log.Fatal("User not found")
	}
	if c, _ := cmd.Flags().GetInt("count"); c > 0 {
		countFlag = uint32(c)
	}
	t := NewTuiModel()
	t.casts.SetResultsCount(countFlag)

	switch {
	case len(parts) == 1 && parts[0] == "casts":
		t.casts.LoadFid(user.Fid)
		t.history.Push(history.Path{
			Type: history.TYPE_LIST,
			Fid:  user.Fid,
		})
		p := tea.NewProgram(t)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
	}
}

func init() {
	rootCmd.AddCommand(exploreCmd)
	exploreCmd.Flags().IntP("count", "c", 0, "Number of casts to show when getting @user/casts")
}

type tuiModel struct {
	casts     *tui2.CastsModel
	cursor    int
	history   *history.History
	statusBar *tui2.StatusBar
}

func NewTuiModel() *tuiModel {
	statusText := "↑/↓/←/→ navigate • q quit"
	m := &tuiModel{
		casts:     tui2.NewCastsModel(),
		history:   history.New(1024),
		statusBar: tui2.NewStatusBar().SetText(statusText).SetHeight(1),
	}
	return m
}

func (t *tuiModel) Init() tea.Cmd {
	return nil
}

func (t *tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	current, err := t.history.Peek()
	if err != nil {
		log.Fatal(err)
	}
	switch msg.(type) {
	case tea.KeyMsg:
		switch msg.(tea.KeyMsg).String() {
		case "ctrl+c", "q":
			cmds = append(cmds, tea.Quit)
		case "up", "k", "down", "j":
			_, cmd := t.casts.Update(msg)
			cmds = append(cmds, cmd)
		case "enter", "right", "l":
			fid, hash, view := t.casts.Status()
			cmd := func() tea.Msg {
				t.statusBar.SetStatus("Loading...")
				t.updateForEnterOrRight(fid, hash, view, current)
				return t.statusBar.SetStatus("")
			}
			cmds = append(cmds, cmd)
		case "left", "h":
			cmd := func() tea.Msg {
				t.statusBar.SetStatus("Loading...")
				t.updateForLeft()
				return t.statusBar.SetStatus("")
			}
			cmds = append(cmds, cmd)
		}
	case tea.WindowSizeMsg:
		h := t.statusBar.Height()
		t.casts.Update(tea.WindowSizeMsg{Width: msg.(tea.WindowSizeMsg).Width, Height: msg.(tea.WindowSizeMsg).Height - h})
		t.statusBar.Update(tea.WindowSizeMsg{Width: msg.(tea.WindowSizeMsg).Width, Height: msg.(tea.WindowSizeMsg).Height})
		/*default:
		_, cmd := t.casts.Update(msg)
		cmds = append(cmds, cmd)
		*/
	}

	return t, tea.Sequence(cmds...)
}

func (t *tuiModel) updateForEnterOrRight(fid uint64, hash []byte, view tui2.View, current history.Path) {

	if current.Type == history.TYPE_LIST {
		t.casts.LoadCasts(fid, hash)
		t.history.SetView(view.Cursor, view.Start, view.End)
		t.history.Push(history.Path{
			Type: history.TYPE_THREAD,
			Fid:  fid,
			Hash: hash,
		})
	} else if current.Type == history.TYPE_THREAD {
		t.handleThreadView(fid, hash, view)
	}
	t.history.SetView(view.Cursor, view.Start, view.End)
}

func (t *tuiModel) handleThreadView(fid uint64, hash []byte, view tui2.View) {
	if t.casts.IsFocus() {
		action := t.casts.GetItemInFocus()
		if action == "" {
			return
		}
		//actionType, nextFid, nextHash := parseAction(action)
		actionType := action[0:3]
		switch actionType {
		case "fid":
			parts := strings.Split(action, ":")
			nextFid, _ := strconv.ParseUint(parts[1], 10, 64)
			t.history.SetView(view.Cursor, view.Start, view.End)
			t.casts.LoadFid(nextFid)
			t.history.Push(history.Path{Type: history.TYPE_LIST, Fid: nextFid})
			t.casts.SetFocus(false, 0)
		case "cst":
			parts := strings.Split(action, ":")
			nextFid, _ := strconv.ParseUint(parts[1], 10, 64)
			nextHash, _ := hex.DecodeString(parts[2])
			t.history.SetView(view.Cursor, view.Start, view.End)
			t.casts.LoadCasts(nextFid, nextHash)
			t.history.Push(history.Path{Type: history.TYPE_THREAD, Fid: nextFid, Hash: nextHash})
		case "url":
			url := action[4:]
			OpenUrl(url)
		}
	} else {
		t.casts.SetFocus(true, view.Cursor)
	}
}

func (t *tuiModel) updateForLeft() {
	t.statusBar.SetStatus("Loading...")
	if _, err := t.history.Pop(); err == nil {
		if prev, err := t.history.Peek(); err == nil {
			switch prev.Type {
			case history.TYPE_LIST:
				t.casts.LoadFid(prev.Fid)
			case history.TYPE_THREAD:
				t.casts.LoadCasts(prev.Fid, prev.Hash)
			}
			t.casts.SetView(tui2.View{Start: prev.ViewStart, End: prev.ViewEnd, Cursor: prev.Cursor})
			t.casts.SetFocus(false, prev.Cursor)

			//t.cursor = prev.Cursor
		}
	}
	t.statusBar.SetStatus("")
}

func (t *tuiModel) View() string {
	current, _ := t.history.Peek()
	var output strings.Builder
	switch current.Type {
	case history.TYPE_LIST:
		output.WriteString(t.casts.View())
	case history.TYPE_THREAD, history.TYPE_CAST:
		output.WriteString(t.casts.View())
	}
	//fid, hash, status := t.casts.Status()
	//t.statusBar.SetStatus(fmt.Sprintf("%d/%x %d %d %d %d", fid, hash, status.Start, status.Cursor, status.End, status.Height))
	output.WriteString(t.statusBar.View())
	return output.String()
}
