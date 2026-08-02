package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Filter  key.Binding
	Doctor  key.Binding
	Profile key.Binding
	Refresh key.Binding
	Quit    key.Binding
	Back    key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter:   key.NewBinding(key.WithKeys("enter", "right"), key.WithHelp("enter/→", "expand/open page")),
		Filter:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Doctor:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "doctor")),
		Profile: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "profile")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh latest versions")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Back:    key.NewBinding(key.WithKeys("esc", "left"), key.WithHelp("esc/←", "back/collapse")),
	}
}
