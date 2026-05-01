package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
	"var-cli/api"
	"var-cli/config"
)

func (m trackerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case configLoadedMsg:
		m.appConfig = config.AppConfig(msg)
		m.apiClient = api.NewClient(m.appConfig.APIToken)
		m.state = stateVerifyingToken
		m.loading = "Verificando token..."
		return m, verifyTokenCmd(m.apiClient)

	case configErrorMsg:
		m.state = stateLogin
		m.err = nil
		return m, m.tokenInput.Focus()

	case profileLoadedMsg:
		m.profile = msg.Profile
		m.state = stateLoadingData
		m.loading = "Cargando datos..."
		return m, loadDataCmd(m.apiClient)

	case profileErrorMsg:
		m.appConfig.APIToken = ""
		m.apiClient = nil
		m.state = stateLogin
		m.err = msg.Err
		m.tokenInput.SetValue("")
		return m, m.tokenInput.Focus()

	case dataLoadedMsg:
		m.entries = msg.Entries
		m.projects = msg.Projects
		m.tags = msg.Tags
		m.state = stateEntries
		m.err = nil
		return m, nil

	case dataErrorMsg:
		m.state = stateEntries
		m.err = msg.Err
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		switch m.state {
		case stateLogin:
			if msg.String() == "ctrl+v" {
				clip, err := clipboard.ReadAll()
				if err == nil && clip != "" {
					m.tokenInput.SetValue(m.tokenInput.Value() + clip)
				}
				return m, nil
			}
			if msg.String() == "enter" {
				token := m.tokenInput.Value()
				if token != "" {
					m.appConfig.APIToken = token
					m.apiClient = api.NewClient(token)
					m.state = stateVerifyingToken
					m.loading = "Verificando token..."
					m.err = nil
					return m, tea.Batch(
						func() tea.Msg {
							_ = config.Save(m.appConfig)
							return nil
						},
						verifyTokenCmd(m.apiClient),
					)
				}
			}
			m.tokenInput, cmd = m.tokenInput.Update(msg)
			return m, cmd

		case stateEntries:
			if msg.String() == "q" {
				return m, tea.Quit
			}
		}
	}

	return m, nil
}
