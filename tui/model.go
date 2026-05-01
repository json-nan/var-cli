package tui

import (
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"var-cli/config"
)

type sessionState int

const (
	stateInitializing sessionState = iota
	stateLogin
	stateMain
)

type configLoadedMsg config.AppConfig
type configErrorMsg struct{}

type trackerModel struct {
	state        sessionState
	appConfig    config.AppConfig
	tokenInput   textinput.Model
	progressBar  progress.Model
	currentHours float64
}

func NewModel() trackerModel {
	ti := textinput.New()
	ti.Placeholder = "Pega tu API Token aquí..."
	ti.Focus()
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'

	return trackerModel{
		state:       stateInitializing,
		tokenInput:  ti,
		progressBar: progress.New(progress.WithDefaultBlend()),
	}
}

func (m trackerModel) Init() tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.Load()
		if err != nil || cfg.APIToken == "" {
			return configErrorMsg{}
		}
		return configLoadedMsg(cfg)
	}
}
