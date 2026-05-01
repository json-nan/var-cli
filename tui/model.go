package tui

import (
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"var-cli/api"
	"var-cli/config"
)

type sessionState int

const (
	stateInitializing sessionState = iota
	stateVerifyingToken
	stateLogin
	stateLoadingData
	stateEntries
)

type configLoadedMsg config.AppConfig
type configErrorMsg struct{}

type profileLoadedMsg struct {
	Profile *api.Profile
}
type profileErrorMsg struct {
	Err error
}

type dataLoadedMsg struct {
	Entries  []api.TimeEntry
	Projects []api.Project
	Tags     []api.Tag
}
type dataErrorMsg struct {
	Err error
}

type trackerModel struct {
	state      sessionState
	appConfig  config.AppConfig
	apiClient  *api.Client
	tokenInput textinput.Model

	profile  *api.Profile
	entries  []api.TimeEntry
	projects []api.Project
	tags     []api.Tag

	err     error
	loading string
}

func NewModel() trackerModel {
	ti := textinput.New()
	ti.Placeholder = "Pega tu API Token aquí..."
	ti.Focus()
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'

	return trackerModel{
		state:      stateInitializing,
		tokenInput: ti,
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

func verifyTokenCmd(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		profile, err := client.GetProfile()
		if err != nil {
			return profileErrorMsg{Err: err}
		}
		return profileLoadedMsg{Profile: profile}
	}
}

func getWeekRange() (string, string) {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := now.AddDate(0, 0, -weekday+1)
	end := start.AddDate(0, 0, 6)
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}

func loadDataCmd(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		startDate, endDate := getWeekRange()

		entries, err := client.GetTimeEntries(startDate, endDate)
		if err != nil {
			return dataErrorMsg{Err: err}
		}

		projects, err := client.GetProjects()
		if err != nil {
			return dataErrorMsg{Err: err}
		}

		tags, err := client.GetTags()
		if err != nil {
			return dataErrorMsg{Err: err}
		}

		return dataLoadedMsg{
			Entries:  entries,
			Projects: projects,
			Tags:     tags,
		}
	}
}
