package tui

import (
	"sort"
	"strconv"
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
	stateFormDate
	stateFormDescription
	stateFormProject
	stateFormTags
	stateFormTime
	stateFormBillable
	stateFormSaving
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

type entryCreatedMsg struct{}
type entryErrorMsg struct {
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

	showAllEntries bool

	frequentProjectCount int
	frequentTagCount     int

	currentVersion string
	latestVersion  string
	latestURL      string
	updateAvailable bool
	updateError     error

	err     error
	loading string

	// Form inputs
	dateInput textinput.Model
	descInput textinput.Model
	timeInput textinput.Model

	// Form selection state
	formProjectCursor int
	formTagCursor     int
	formSelectedTags  map[int]bool // tagID -> selected
	formBillable      bool
}

func NewModel(version string) trackerModel {
	ti := textinput.New()
	ti.Placeholder = "Pega tu API Token aquí..."
	ti.Focus()
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'

	now := time.Now().Format("2006-01-02")

	dateIn := textinput.New()
	dateIn.Placeholder = now
	dateIn.SetValue(now)

	descIn := textinput.New()
	descIn.Placeholder = "Descripción del trabajo realizado..."

	timeIn := textinput.New()
	timeIn.Placeholder = "60"

	return trackerModel{
		state:          stateInitializing,
		tokenInput:     ti,
		dateInput:      dateIn,
		descInput:      descIn,
		timeInput:      timeIn,
		currentVersion: version,
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

func getTwoWeekRange() (string, string) {
	now := time.Now()
	start := now.AddDate(0, 0, -13)
	return start.Format("2006-01-02"), now.Format("2006-01-02")
}

func loadDataCmd(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		startDate, endDate := getTwoWeekRange()

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

func createEntryCmd(client *api.Client, entry api.NewTimeEntry) tea.Cmd {
	return func() tea.Msg {
		_, err := client.CreateTimeEntry(entry)
		if err != nil {
			return entryErrorMsg{Err: err}
		}
		return entryCreatedMsg{}
	}
}

func (m *trackerModel) resetForm() {
	m.dateInput.SetValue(time.Now().Format("2006-01-02"))
	m.descInput.SetValue("")
	m.timeInput.SetValue("")
	m.formProjectCursor = 0
	m.formTagCursor = 0
	m.formSelectedTags = make(map[int]bool)
	m.formBillable = false
	m.updateError = nil
}

func (m *trackerModel) computeFrequencies() {
	projectCounts := make(map[int]int)
	for _, e := range m.entries {
		projectCounts[e.Project.ID]++
	}

	sort.Slice(m.projects, func(i, j int) bool {
		ci := projectCounts[m.projects[i].ID]
		cj := projectCounts[m.projects[j].ID]
		if ci != cj {
			return ci > cj
		}
		return m.projects[i].Name < m.projects[j].Name
	})

	m.frequentProjectCount = 0
	for _, p := range m.projects {
		if projectCounts[p.ID] > 0 {
			m.frequentProjectCount++
		}
	}

	tagCounts := make(map[int]int)
	for _, e := range m.entries {
		for _, t := range e.Tags {
			tagCounts[t.ID]++
		}
	}

	sort.Slice(m.tags, func(i, j int) bool {
		ci := tagCounts[m.tags[i].ID]
		cj := tagCounts[m.tags[j].ID]
		if ci != cj {
			return ci > cj
		}
		return m.tags[i].Name < m.tags[j].Name
	})

	m.frequentTagCount = 0
	for _, t := range m.tags {
		if tagCounts[t.ID] > 0 {
			m.frequentTagCount++
		}
	}
}

func (m trackerModel) displayEntries() []api.TimeEntry {
	if m.showAllEntries {
		return m.entries
	}
	cutoff := time.Now().AddDate(0, 0, -6).Format("2006-01-02")
	var filtered []api.TimeEntry
	for _, e := range m.entries {
		if e.Date >= cutoff {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func (m *trackerModel) buildNewEntry() (api.NewTimeEntry, error) {
	minutes, err := strconv.Atoi(m.timeInput.Value())
	if err != nil {
		return api.NewTimeEntry{}, err
	}

	var tagIDs []int
	for id := range m.formSelectedTags {
		tagIDs = append(tagIDs, id)
	}

	projectID := 0
	if len(m.projects) > 0 {
		projectID = m.projects[m.formProjectCursor].ID
	}

	return api.NewTimeEntry{
		Date:        m.dateInput.Value(),
		Description: m.descInput.Value(),
		ProjectID:   projectID,
		TagIDs:      tagIDs,
		Minutes:     minutes,
		IsBillable:  m.formBillable,
	}, nil
}
