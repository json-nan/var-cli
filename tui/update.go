package tui

import (
	"fmt"
	"sort"
	"strconv"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
	"var-cli/api"
	"var-cli/config"
)

type updateCheckMsg struct {
	Info *api.ReleaseInfo
	Err  error
}

type updateAppliedMsg struct{}

type updateErrorMsg struct {
	Err error
}

func checkForUpdateCmd(version string) tea.Cmd {
	return func() tea.Msg {
		info, err := api.CheckForUpdate(version)
		return updateCheckMsg{Info: info, Err: err}
	}
}

func applyUpdateCmd(url string) tea.Cmd {
	return func() tea.Msg {
		err := api.ApplyUpdate(url)
		if err != nil {
			return updateErrorMsg{Err: err}
		}
		return updateAppliedMsg{}
	}
}

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
		sort.Slice(m.entries, func(i, j int) bool {
			if m.entries[i].Date != m.entries[j].Date {
				return m.entries[i].Date > m.entries[j].Date
			}
			return m.entries[i].ID > m.entries[j].ID
		})
		m.projects = msg.Projects
		m.tags = msg.Tags
		m.computeFrequencies()
		m.err = nil
		m.entryCursor = 0

		// Check if we should show "What's New" after an update
		if m.currentVersion != "dev" && m.currentVersion != "" &&
			m.appConfig.LastVersion != "" &&
			m.appConfig.LastVersion != m.currentVersion {
			m.changelogChanges = getChangesSince(m.changelog, m.appConfig.LastVersion, m.currentVersion)
			if len(m.changelogChanges) > 0 {
				m.state = stateWhatsNew
				m.appConfig.LastVersion = m.currentVersion
				_ = config.Save(m.appConfig)
				return m, checkForUpdateCmd(m.currentVersion)
			}
		}
		// First run: store current version without showing changelog
		if m.currentVersion != "dev" && m.currentVersion != "" && m.appConfig.LastVersion == "" {
			m.appConfig.LastVersion = m.currentVersion
			_ = config.Save(m.appConfig)
		}
		m.state = stateEntries
		return m, checkForUpdateCmd(m.currentVersion)

	case dataErrorMsg:
		m.state = stateEntries
		m.err = msg.Err
		return m, nil

	case entryCreatedMsg:
		m.quickReset()
		m.state = stateFormDescription
		m.descInput.Focus()
		return m, loadDataCmd(m.apiClient)

	case entryUpdatedMsg:
		m.editingEntry = nil
		m.resetForm()
		m.state = stateEntries
		m.flash = "Entrada actualizada."
		return m, loadDataCmd(m.apiClient)

	case entryErrorMsg:
		m.editingEntry = nil
		m.err = msg.Err
		m.state = stateEntries
		return m, nil

	case entryDeletedMsg:
		m.state = stateLoadingData
		m.loading = "Recargando entradas..."
		m.entryCursor = 0
		return m, loadDataCmd(m.apiClient)

	case deleteErrorMsg:
		m.err = msg.Err
		m.state = stateEntries
		return m, nil

	case updateCheckMsg:
		if msg.Err != nil {
			m.updateError = msg.Err
		} else if msg.Info != nil {
			m.latestVersion = msg.Info.Version
			m.latestURL = msg.Info.URL
			m.updateAvailable = true
		}
		return m, nil

	case updateAppliedMsg:
		m.state = stateEntries
		m.updateAvailable = false
		m.err = nil
		m.loading = "Actualizado. Reinicia para usar la nueva version."
		return m, tea.Println("Actualizado. Reinicia para usar la nueva version.")

	case updateErrorMsg:
		m.updateError = msg.Err
		m.state = stateEntries
		return m, nil

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.dayProgress.SetWidth(max(20, msg.Width-10))
		m.weekProgress.SetWidth(max(20, msg.Width-10))
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
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "n":
				m.resetForm()
				m.state = stateFormDate
				m.dateInput.Focus()
				return m, nil
			case "e":
				disp := m.displayEntries()
				if m.entryCursor < len(disp) {
					entry := disp[m.entryCursor]
					if !entry.IsEditable {
						m.flash = "Esta entrada no se puede editar."
						return m, nil
					}
					m.editingEntry = &entry
					m.dateInput.SetValue(entry.Date)
					m.descInput.SetValue(entry.Description)
					for i, p := range m.projects {
						if p.ID == entry.Project.ID {
							m.formProjectCursor = i
							break
						}
					}
					m.formSelectedTags = make(map[int]bool)
					for _, t := range entry.Tags {
						m.formSelectedTags[t.ID] = true
					}
					m.timeInput.SetValue(formatTimeInput(entry.Minutes))
					m.formBillable = entry.IsBillable
					m.state = stateFormDate
					m.dateInput.Focus()
				}
				return m, nil
			case "a":
				m.showAllEntries = !m.showAllEntries
				m.entryCursor = 0
				return m, nil
			case "r":
				m.state = stateLoadingData
				m.loading = "Recargando..."
				return m, loadDataCmd(m.apiClient)
			case "u":
				if m.updateAvailable && m.latestURL != "" {
					m.state = stateLoadingData
					m.loading = fmt.Sprintf("Actualizando a %s...", m.latestVersion)
					return m, applyUpdateCmd(m.latestURL)
				}
				m.loading = "Buscando actualizaciones..."
				return m, checkForUpdateCmd(m.currentVersion)
			case "d":
				if len(m.displayEntries()) > 0 {
					m.state = stateDeletingConfirm
					m.deleteConfirm = false
				}
				return m, nil
			case "c":
				m.state = stateChangelog
				return m, nil
			case "up", "k":
				if m.entryCursor > 0 {
					m.entryCursor--
				}
				return m, nil
			case "down", "j":
				disp := m.displayEntries()
				if m.entryCursor < len(disp)-1 {
					m.entryCursor++
				}
				return m, nil
			}

		case stateDeletingConfirm:
			switch msg.String() {
			case "esc", "q":
				m.state = stateEntries
				return m, nil
			case "left", "h":
				m.deleteConfirm = false
			case "right", "l":
				m.deleteConfirm = true
			case "enter":
				if m.deleteConfirm {
					disp := m.displayEntries()
					if m.entryCursor < len(disp) {
						entry := disp[m.entryCursor]
						m.state = stateDeleting
						m.loading = "Eliminando entrada..."
						return m, deleteEntryCmd(m.apiClient, entry.ID)
					}
				}
				m.state = stateEntries
				return m, nil
			}
			return m, nil

		case stateFormDate:
			m.flash = ""
			if msg.String() == "esc" {
				m.state = stateEntries
				return m, nil
			}
			emptyDays := m.emptyDays()
			switch msg.String() {
			case "enter":
				m.state = stateFormDescription
				m.descInput.Focus()
				return m, nil
			case "up", "k":
				if len(emptyDays) > 0 {
					if m.formEmptyDayCursor < 0 {
						m.formEmptyDayCursor = 0
					} else if m.formEmptyDayCursor > 0 {
						m.formEmptyDayCursor--
					}
					m.dateInput.SetValue(emptyDays[m.formEmptyDayCursor])
				}
				return m, nil
			case "down", "j":
				if len(emptyDays) > 0 {
					if m.formEmptyDayCursor < len(emptyDays)-1 {
						m.formEmptyDayCursor++
					} else if m.formEmptyDayCursor < 0 {
						m.formEmptyDayCursor = 0
					}
					m.dateInput.SetValue(emptyDays[m.formEmptyDayCursor])
				}
				return m, nil
			}
			m.dateInput, cmd = m.dateInput.Update(msg)
			return m, cmd

		case stateFormDescription:
			m.flash = ""
			if msg.String() == "esc" {
				if m.formDescSuggestionCursor >= 0 {
					m.formDescSuggestionCursor = -1
					return m, nil
				}
				m.state = stateFormDate
				m.dateInput.Focus()
				return m, nil
			}
			suggestions := m.descSuggestions()
			switch msg.String() {
			case "up", "k":
				if len(suggestions) > 0 {
					if m.formDescSuggestionCursor > 0 {
						m.formDescSuggestionCursor--
					} else if m.formDescSuggestionCursor < 0 {
						m.formDescSuggestionCursor = len(suggestions) - 1
					}
				}
				return m, nil
			case "down", "j":
				if len(suggestions) > 0 {
					if m.formDescSuggestionCursor < len(suggestions)-1 {
						m.formDescSuggestionCursor++
					} else if m.formDescSuggestionCursor < 0 {
						m.formDescSuggestionCursor = 0
					}
				}
				return m, nil
			case "enter":
				if m.formDescSuggestionCursor >= 0 && m.formDescSuggestionCursor < len(suggestions) {
					// Apply suggestion but follow normal flow
					e := suggestions[m.formDescSuggestionCursor]
					m.descInput.SetValue(e.Description)
					for i, p := range m.projects {
						if p.ID == e.Project.ID {
							m.formProjectCursor = i
							break
						}
					}
					m.formSelectedTags = make(map[int]bool)
					for _, t := range e.Tags {
						m.formSelectedTags[t.ID] = true
					}
					m.timeInput.SetValue(strconv.Itoa(e.Minutes))
					m.formBillable = e.IsBillable
				}
				m.formDescSuggestionCursor = -1
				if len(m.projects) == 0 {
					if len(m.tags) == 0 {
						m.state = stateFormTime
						m.timeInput.Focus()
					} else {
						m.state = stateFormTags
					}
				} else {
					m.state = stateFormProject
				}
				return m, nil
			}
			m.descInput, cmd = m.descInput.Update(msg)
			m.formDescSuggestionCursor = -1
			return m, cmd

		case stateFormProject:
			m.flash = ""
			if msg.String() == "esc" {
				m.state = stateFormDescription
				m.descInput.Focus()
				return m, nil
			}
			switch msg.String() {
			case "up", "k":
				if m.formProjectCursor > 0 {
					m.formProjectCursor--
				}
			case "down", "j":
				if m.formProjectCursor < len(m.projects)-1 {
					m.formProjectCursor++
				}
			case "enter":
				if len(m.tags) == 0 {
					m.state = stateFormTime
					m.timeInput.Focus()
				} else {
					m.state = stateFormTags
				}
			}
			return m, nil

		case stateFormTags:
			m.flash = ""
			if msg.String() == "esc" {
				m.state = stateFormProject
				return m, nil
			}
			switch msg.String() {
			case "up", "k":
				if m.formTagCursor > 0 {
					m.formTagCursor--
				}
			case "down", "j":
				if m.formTagCursor < len(m.tags)-1 {
					m.formTagCursor++
				}
			case " ", "space":
				if len(m.tags) > 0 {
					tagID := m.tags[m.formTagCursor].ID
					if m.formSelectedTags[tagID] {
						delete(m.formSelectedTags, tagID)
					} else {
						if m.formSelectedTags == nil {
							m.formSelectedTags = make(map[int]bool)
						}
						m.formSelectedTags[tagID] = true
					}
				}
			case "enter":
				m.state = stateFormTime
				m.timeInput.Focus()
			}
			return m, nil

		case stateFormTime:
			m.flash = ""
			if msg.String() == "esc" {
				if len(m.tags) > 0 {
					m.state = stateFormTags
				} else if len(m.projects) > 0 {
					m.state = stateFormProject
				} else {
					m.state = stateFormDescription
					m.descInput.Focus()
				}
				return m, nil
			}
			if msg.String() == "enter" {
				m.state = stateFormBillable
				return m, nil
			}
			m.timeInput, cmd = m.timeInput.Update(msg)
			return m, cmd

		case stateFormBillable:
			m.flash = ""
			if msg.String() == "esc" {
				m.state = stateFormTime
				m.timeInput.Focus()
				return m, nil
			}
			switch msg.String() {
			case "left", "h":
				m.formBillable = false
			case "right", "l":
				m.formBillable = true
			case "enter":
				entry, err := m.buildNewEntry()
				if err != nil {
					m.err = err
					m.state = stateEntries
					return m, nil
				}
				if m.editingEntry != nil {
					m.state = stateFormSaving
					m.loading = "Actualizando entrada..."
					return m, updateEntryCmd(m.apiClient, m.editingEntry.ID, entry)
				}
				m.state = stateFormSaving
				m.loading = "Guardando entrada..."
				return m, createEntryCmd(m.apiClient, entry)
			}
			return m, nil

		case stateChangelog:
			switch msg.String() {
			case "q", "esc":
				m.state = stateEntries
			}
			return m, nil

		case stateWhatsNew:
			switch msg.String() {
			case "q", "esc", "enter":
				m.state = stateEntries
			}
			return m, nil
		}
	}

	return m, nil
}
