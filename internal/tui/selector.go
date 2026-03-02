package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	gh "github.com/yukikotani231/gh-pr-review/internal/github"
)

// prItem は list.DefaultItem インターフェースを実装する
type prItem struct {
	number    int
	title     string
	author    string
	updatedAt string
	isDraft   bool
}

func (i prItem) Title() string {
	if i.isDraft {
		return fmt.Sprintf("#%d [draft] %s", i.number, i.title)
	}
	return fmt.Sprintf("#%d %s", i.number, i.title)
}

func (i prItem) Description() string {
	return fmt.Sprintf("by @%s  updated %s", i.author, formatTime(i.updatedAt))
}

func (i prItem) FilterValue() string {
	return fmt.Sprintf("#%d %s %s", i.number, i.title, i.author)
}

// --- メッセージ型 ---

type openPRsFetchedMsg struct {
	PRs []gh.PRListItem
	Err error
}

// --- SelectorModel ---

// SelectorModel はPR選択画面のモデル
type SelectorModel struct {
	client   *gh.Client
	list     list.Model
	selected int // 選択されたPR番号。0なら未選択
	quitting bool
	err      error
}

// NewSelectorModel は新しいSelectorModelを返す
func NewSelectorModel(client *gh.Client) SelectorModel {
	delegate := list.NewDefaultDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.Title = "PRを選択してレビュー"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	return SelectorModel{
		client: client,
		list:   l,
	}
}

func (m SelectorModel) Init() tea.Cmd {
	return m.fetchOpenPRsCmd()
}

func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		// フィルタリング中は list にキー入力を委譲
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if item, ok := m.list.SelectedItem().(prItem); ok {
				m.selected = item.number
				return m, tea.Quit
			}
		}

	case openPRsFetchedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		items := make([]list.Item, len(msg.PRs))
		for i, pr := range msg.PRs {
			items[i] = prItem{
				number:    pr.Number,
				title:     pr.Title,
				author:    pr.Author,
				updatedAt: pr.UpdatedAt,
				isDraft:   pr.IsDraft,
			}
		}
		cmd := m.list.SetItems(items)
		return m, cmd
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m SelectorModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  エラー: %v\n\n  qで終了\n", m.err)
	}
	return m.list.View()
}

// Selected は選択されたPR番号を返す。未選択なら0。
func (m SelectorModel) Selected() int {
	return m.selected
}

// Quitting はユーザがquitしたかを返す
func (m SelectorModel) Quitting() bool {
	return m.quitting
}

func (m SelectorModel) fetchOpenPRsCmd() tea.Cmd {
	if m.client == nil {
		return nil
	}
	return func() tea.Msg {
		prs, err := m.client.FetchOpenPRs()
		return openPRsFetchedMsg{PRs: prs, Err: err}
	}
}
