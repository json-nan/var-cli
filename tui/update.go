package tui

import (
	tea "charm.land/bubbletea/v2"
	"var-cli/config"
)

func (m trackerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case configLoadedMsg:
		m.appConfig = config.AppConfig(msg)
		m.state = stateMain
		return m, nil

	case configErrorMsg:
		m.state = stateLogin
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		switch m.state {
		case stateLogin:
			if msg.String() == "enter" {
				token := m.tokenInput.Value()
				if token != "" {
					m.appConfig.APIToken = token
					_ = config.Save(m.appConfig)
					m.state = stateMain
					return m, nil
				}
			}
			m.tokenInput, cmd = m.tokenInput.Update(msg)
			return m, cmd

		case stateMain:
			if msg.String() == "q" {
				return m, tea.Quit
			}
			if msg.String() == "a" {
				m.currentHours += 1.0
				if m.currentHours > 40 {
					m.currentHours = 40
				}
			}
		}
	}

	return m, nil
}
