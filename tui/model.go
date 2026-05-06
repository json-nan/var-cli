package tui

import (
	"sort"
	"strconv"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"var-cli/api"
	"var-cli/config"
)

const (
	targetWeekMinutes = 2640 // 44 hours (Mon 9h + Tue 9h + Wed 9h + Thu 9h + Fri 8h)
)

func targetMinutesForWeekday(wd time.Weekday) int {
	switch wd {
	case time.Monday, time.Tuesday, time.Wednesday, time.Thursday:
		return 540 // 9 hours
	case time.Friday:
		return 480 // 8 hours
	default:
		return 0 // weekends not tracked
	}
}

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
	stateDeletingConfirm
	stateDeleting
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

type entryDeletedMsg struct{}
type deleteErrorMsg struct {
	Err error
}

type trackerModel struct {
	state     sessionState
	appConfig config.AppConfig
	apiClient *api.Client

	// Inputs
	tokenInput textinput.Model
	dateInput  textinput.Model
	descInput  textinput.Model
	timeInput  textinput.Model

	// Spinner for loading states
	spinner spinner.Model

	// Progress bars
	dayProgress  progress.Model
	weekProgress progress.Model

	// Data
	profile  *api.Profile
	entries  []api.TimeEntry
	projects []api.Project
	tags     []api.Tag

	// View state
	showAllEntries bool
	entryCursor    int // for deletion selection
	deleteConfirm  bool

	// Form state
	formProjectCursor int
	formTagCursor     int
	formSelectedTags  map[int]bool
	formBillable      bool

	// Frequencies
	frequentProjectCount int
	frequentTagCount     int

	// Version / update
	currentVersion  string
	latestVersion   string
	latestURL       string
	updateAvailable bool
	updateError     error

	// Terminal width
	width int

	// Error / loading
	err     error
	loading string
}

func NewModel(version string) trackerModel {
	s := spinner.New(spinner.WithSpinner(spinner.Line))

	dp := progress.New(progress.WithWidth(40), progress.WithDefaultBlend(), progress.WithoutPercentage())
	wp := progress.New(progress.WithWidth(40), progress.WithDefaultBlend(), progress.WithoutPercentage())

	ti := textinput.New()
	ti.Placeholder = "Pega tu API Token aqui..."
	ti.Focus()
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'

	now := time.Now().Format("2006-01-02")

	dateIn := textinput.New()
	dateIn.Placeholder = now
	dateIn.SetValue(now)

	descIn := textinput.New()
	descIn.Placeholder = "Descripcion del trabajo realizado..."

	timeIn := textinput.New()
	timeIn.Placeholder = "60"

	return trackerModel{
		state:          stateInitializing,
		tokenInput:     ti,
		dateInput:      dateIn,
		descInput:      descIn,
		timeInput:      timeIn,
		spinner:        s,
		dayProgress:    dp,
		weekProgress:   wp,
		currentVersion: version,
	}
}

func (m trackerModel) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return m.spinner.Tick() },
		func() tea.Msg {
			cfg, err := config.Load()
			if err != nil || cfg.APIToken == "" {
				return configErrorMsg{}
			}
			return configLoadedMsg(cfg)
		},
	)
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

	// Days since Monday of current week
	daysSinceMonday := int(now.Weekday()) - int(time.Monday)
	if daysSinceMonday < 0 {
		daysSinceMonday += 7
	}

	mondayCurrent := now.AddDate(0, 0, -daysSinceMonday)
	start := mondayCurrent.AddDate(0, 0, -7) // Monday of past week
	end := mondayCurrent.AddDate(0, 0, 6)    // Sunday of current week

	return start.Format("2006-01-02"), end.Format("2006-01-02")
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

func deleteEntryCmd(client *api.Client, id int) tea.Cmd {
	return func() tea.Msg {
		err := client.DeleteTimeEntry(id)
		if err != nil {
			return deleteErrorMsg{Err: err}
		}
		return entryDeletedMsg{}
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
