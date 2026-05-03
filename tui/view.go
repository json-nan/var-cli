package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"var-cli/api"
)

var (
	// Base colors
	colorPrimary   = lipgloss.Color("#7D56F4")
	colorSecondary = lipgloss.Color("#04B575")
	colorDanger    = lipgloss.Color("#FF5F5F")
	colorWarning   = lipgloss.Color("#F4D03F")
	colorMuted     = lipgloss.Color("#626262")
	colorFg        = lipgloss.Color("#E0E0E0")
	colorBorder    = lipgloss.Color("#333333")
	colorHighlight = lipgloss.Color("#FF7CCB")

	// Styles
	titleStyle         = lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).MarginBottom(1)
	subtitleStyle      = lipgloss.NewStyle().Bold(true).Foreground(colorFg).MarginBottom(1)
	labelStyle         = lipgloss.NewStyle().Foreground(colorMuted)
	mutedStyle         = lipgloss.NewStyle().Foreground(colorMuted)
	dateHeaderStyle    = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	errorStyle         = lipgloss.NewStyle().Foreground(colorDanger).Bold(true)
	helpStyle          = lipgloss.NewStyle().Foreground(colorMuted)
	billableStyle      = lipgloss.NewStyle().Foreground(colorSecondary)
	selectedStyle      = lipgloss.NewStyle().Background(colorPrimary).Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	cursorStyle        = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	sectionStyle       = lipgloss.NewStyle().Foreground(colorMuted).Bold(true)
	tagCheckStyle      = lipgloss.NewStyle().Foreground(colorSecondary).Bold(true)
	updateStyle        = lipgloss.NewStyle().Foreground(colorWarning).Bold(true)
	boxStyle           = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorBorder).Padding(1, 2)
	progressLabelStyle = lipgloss.NewStyle().Foreground(colorFg).Bold(true)
)

func formatMinutes(minutes int) string {
	h := minutes / 60
	m := minutes % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	} else if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}

func dayName(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	switch t.Weekday() {
	case time.Monday:
		return "Lun"
	case time.Tuesday:
		return "Mar"
	case time.Wednesday:
		return "Mie"
	case time.Thursday:
		return "Jue"
	case time.Friday:
		return "Vie"
	case time.Saturday:
		return "Sab"
	case time.Sunday:
		return "Dom"
	}
	return dateStr
}

func (m trackerModel) getTagNames(tags []api.Tag) string {
	var names []string
	for _, t := range tags {
		names = append(names, t.Name)
	}
	if len(names) == 0 {
		return ""
	}
	return strings.Join(names, ", ")
}

func totalMinutes(entries []api.TimeEntry) int {
	total := 0
	for _, e := range entries {
		total += e.Minutes
	}
	return total
}

func billableMinutes(entries []api.TimeEntry) int {
	total := 0
	for _, e := range entries {
		if e.IsBillable {
			total += e.Minutes
		}
	}
	return total
}

func todayMinutes(entries []api.TimeEntry) int {
	today := time.Now().Format("2006-01-02")
	total := 0
	for _, e := range entries {
		if e.Date == today {
			total += e.Minutes
		}
	}
	return total
}

func weekMinutes(entries []api.TimeEntry) int {
	now := time.Now()
	weekStart := now.AddDate(0, 0, -int(now.Weekday())+1)
	if now.Weekday() == time.Sunday {
		weekStart = now.AddDate(0, 0, -6)
	}
	startStr := weekStart.Format("2006-01-02")
	total := 0
	for _, e := range entries {
		if e.Date >= startStr {
			total += e.Minutes
		}
	}
	return total
}

func (m trackerModel) View() tea.View {
	var s string

	switch m.state {
	case stateInitializing:
		s = m.renderLoading("Iniciando...")

	case stateVerifyingToken:
		s = m.renderLoading("Verificando token...")

	case stateLogin:
		s = m.renderLogin()

	case stateLoadingData:
		s = m.renderLoading(m.loading)

	case stateEntries:
		s = m.renderEntriesView()

	case stateDeletingConfirm:
		s = m.renderDeleteConfirm()

	case stateDeleting:
		s = m.renderLoading("Eliminando entrada...")

	case stateFormDate:
		s = m.renderFormDate()
	case stateFormDescription:
		s = m.renderFormDescription()
	case stateFormProject:
		s = m.renderFormProject()
	case stateFormTags:
		s = m.renderFormTags()
	case stateFormTime:
		s = m.renderFormTime()
	case stateFormBillable:
		s = m.renderFormBillable()
	case stateFormSaving:
		s = m.renderLoading("Guardando entrada...")
	}

	finalView := lipgloss.NewStyle().Margin(1, 2).Render(s)
	return tea.NewView(finalView)
}

func (m trackerModel) renderLoading(text string) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("VAR CLI") + "\n\n")
	b.WriteString(m.spinner.View() + " " + text)
	return b.String()
}

func (m trackerModel) renderLogin() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Autenticacion Requerida") + "\n\n")
	if m.err != nil {
		b.WriteString(errorStyle.Render("Token invalido o expirado. Intenta de nuevo.") + "\n\n")
	}
	b.WriteString("Ingresa tu token de API para continuar:\n\n")
	b.WriteString(m.tokenInput.View() + "\n\n")
	b.WriteString(helpStyle.Render("Ctrl+V para pegar  Enter para guardar"))
	return b.String()
}

func (m trackerModel) renderEntriesView() string {
	var b strings.Builder
	entries := m.displayEntries()

	// Header with profile
	b.WriteString(m.renderHeader() + "\n")

	// Update banner
	if m.updateAvailable {
		b.WriteString(updateStyle.Render(fmt.Sprintf("v%s disponible -- presiona 'u' para actualizar", m.latestVersion)) + "\n")
	} else if m.updateError != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error al buscar actualizaciones: %v", m.updateError)) + "\n")
	}

	// Progress section
	b.WriteString(m.renderProgressSection() + "\n")

	// Filter label
	if m.showAllEntries {
		b.WriteString(labelStyle.Render("Mostrando: ultimas 2 semanas") + "\n")
	} else {
		b.WriteString(labelStyle.Render("Mostrando: ultimos 7 dias") + "\n")
	}

	// Error
	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n")
	}

	// Entries
	if len(entries) == 0 {
		b.WriteString("\nNo hay entradas registradas en este periodo.\n")
	} else {
		b.WriteString("\n")
		entriesByDate := groupEntriesByDate(entries)
		dates := sortedDates(entriesByDate)
		globalIdx := 0
		for _, date := range dates {
			dayEntries := entriesByDate[date]
			b.WriteString(m.renderDateHeader(date, dayEntries) + "\n")
			for _, e := range dayEntries {
				b.WriteString(m.renderEntry(e, globalIdx == m.entryCursor) + "\n")
				globalIdx++
			}
			b.WriteString("\n")
		}
	}

	// Help bar
	b.WriteString(m.renderHelpBar())
	return b.String()
}

func (m trackerModel) renderHeader() string {
	var parts []string
	if m.profile != nil && m.profile.Name != "" {
		parts = append(parts, m.profile.Name)
		if m.profile.Position != "" {
			parts = append(parts, m.profile.Position)
		}
	}
	if len(parts) == 0 {
		return titleStyle.Render("VAR CLI")
	}
	return titleStyle.Render(strings.Join(parts, " / "))
}

func (m trackerModel) renderProgressSection() string {
	var b strings.Builder

	todayMin := todayMinutes(m.entries)
	weekMin := weekMinutes(m.entries)
	billMin := billableMinutes(m.entries)
	totalMin := totalMinutes(m.entries)

	// Day progress
	dayPct := float64(todayMin) / float64(targetDayMinutes)
	if dayPct > 1 {
		dayPct = 1
	}
	dayBar := m.dayProgress.ViewAs(dayPct)
	dayLabel := fmt.Sprintf("Hoy:  %s / %s", formatMinutes(todayMin), formatMinutes(targetDayMinutes))
	if todayMin > targetDayMinutes {
		dayLabel = fmt.Sprintf("Hoy:  %s / %s  (%s extra)", formatMinutes(todayMin), formatMinutes(targetDayMinutes), formatMinutes(todayMin-targetDayMinutes))
	}
	b.WriteString(progressLabelStyle.Render(dayLabel) + "\n")
	b.WriteString(dayBar + "\n")

	// Week progress
	weekPct := float64(weekMin) / float64(targetWeekMinutes)
	if weekPct > 1 {
		weekPct = 1
	}
	weekBar := m.weekProgress.ViewAs(weekPct)
	weekLabel := fmt.Sprintf("Sem:  %s / %s", formatMinutes(weekMin), formatMinutes(targetWeekMinutes))
	if weekMin > targetWeekMinutes {
		weekLabel = fmt.Sprintf("Sem:  %s / %s  (%s extra)", formatMinutes(weekMin), formatMinutes(targetWeekMinutes), formatMinutes(weekMin-targetWeekMinutes))
	}
	b.WriteString(progressLabelStyle.Render(weekLabel) + "\n")
	b.WriteString(weekBar + "\n")

	// Billable summary
	if billMin > 0 {
		b.WriteString(billableStyle.Render(fmt.Sprintf("Facturable: %s  Total: %s", formatMinutes(billMin), formatMinutes(totalMin))) + "\n")
	}

	return b.String()
}

func (m trackerModel) renderDateHeader(date string, entries []api.TimeEntry) string {
	dayTotal := 0
	for _, e := range entries {
		dayTotal += e.Minutes
	}
	dn := dayName(date)
	today := time.Now().Format("2006-01-02")
	isToday := date == today

	text := fmt.Sprintf("%s  %s  (%s)", dn, date, formatMinutes(dayTotal))
	if isToday {
		return dateHeaderStyle.Render("[ " + text + " ]")
	}
	return dateHeaderStyle.Render("  " + text)
}

func (m trackerModel) renderEntry(e api.TimeEntry, selected bool) string {
	var parts []string

	desc := e.Description
	if len(desc) > 50 {
		desc = desc[:47] + "..."
	}

	cursor := "   "
	if selected {
		cursor = cursorStyle.Render(" > ")
	}

	parts = append(parts, fmt.Sprintf("%s%s", cursor, desc))
	parts = append(parts, mutedStyle.Render(fmt.Sprintf("[%s]", e.Project.Name)))
	parts = append(parts, lipgloss.NewStyle().Foreground(colorWarning).Render(formatMinutes(e.Minutes)))
	if e.IsBillable {
		parts = append(parts, billableStyle.Render("$"))
	}
	tagNames := m.getTagNames(e.Tags)
	if tagNames != "" {
		parts = append(parts, mutedStyle.Render(fmt.Sprintf("(%s)", tagNames)))
	}
	return strings.Join(parts, " ")
}

func (m trackerModel) renderDeleteConfirm() string {
	var b strings.Builder
	disp := m.displayEntries()
	if m.entryCursor >= len(disp) {
		return ""
	}
	entry := disp[m.entryCursor]

	b.WriteString(titleStyle.Render("Eliminar Entrada") + "\n\n")
	b.WriteString(fmt.Sprintf("Descripcion: %s\n", entry.Description))
	b.WriteString(fmt.Sprintf("Fecha:       %s\n", entry.Date))
	b.WriteString(fmt.Sprintf("Duracion:    %s\n", formatMinutes(entry.Minutes)))
	b.WriteString(fmt.Sprintf("Proyecto:    %s\n", entry.Project.Name))
	b.WriteString("\n")

	noStyle := labelStyle
	yesStyle := labelStyle
	if m.deleteConfirm {
		yesStyle = selectedStyle
	} else {
		noStyle = selectedStyle
	}

	b.WriteString(noStyle.Render("  Cancelar  ") + "    " + yesStyle.Render("  Eliminar  ") + "\n")
	b.WriteString("\n" + helpStyle.Render("<-- --> para cambiar  Enter para confirmar  Esc para cancelar"))
	return b.String()
}

func (m trackerModel) renderHelpBar() string {
	var keys []string
	keys = append(keys, "n nuevo")
	keys = append(keys, "d eliminar")
	if m.showAllEntries {
		keys = append(keys, "a semana")
	} else {
		keys = append(keys, "a todo")
	}
	keys = append(keys, "r recargar")
	if m.updateAvailable {
		keys = append(keys, "u actualizar")
	}
	keys = append(keys, "q salir")
	return helpStyle.Render(strings.Join(keys, "  "))
}

func groupEntriesByDate(entries []api.TimeEntry) map[string][]api.TimeEntry {
	grouped := make(map[string][]api.TimeEntry)
	for _, e := range entries {
		grouped[e.Date] = append(grouped[e.Date], e)
	}
	return grouped
}

func sortedDates(grouped map[string][]api.TimeEntry) []string {
	var dates []string
	for d := range grouped {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	return dates
}

// ---- Form views ----

func formHeader(step, total int, title string) string {
	progress := fmt.Sprintf("Paso %d/%d", step, total)
	return titleStyle.Render(title) + "  " + labelStyle.Render(progress) + "\n\n"
}

func formFooter() string {
	return "\n" + helpStyle.Render("Enter continuar  Esc cancelar")
}

func (m trackerModel) renderFormDate() string {
	var b strings.Builder
	b.WriteString(formHeader(1, 6, "Nueva Entrada"))
	b.WriteString("Fecha (YYYY-MM-DD):\n\n")
	b.WriteString(m.dateInput.View() + "\n")
	b.WriteString(formFooter())
	return b.String()
}

func (m trackerModel) renderFormDescription() string {
	var b strings.Builder
	b.WriteString(formHeader(2, 6, "Nueva Entrada"))
	b.WriteString("Descripcion:\n\n")
	b.WriteString(m.descInput.View() + "\n")
	b.WriteString(formFooter())
	return b.String()
}

func (m trackerModel) renderFormProject() string {
	var b strings.Builder
	b.WriteString(formHeader(3, 6, "Nueva Entrada"))
	b.WriteString("Selecciona el proyecto:\n\n")

	if len(m.projects) == 0 {
		b.WriteString(labelStyle.Render("No hay proyectos disponibles.") + "\n")
	} else {
		if m.frequentProjectCount > 0 {
			b.WriteString(sectionStyle.Render("- Frecuentes") + "\n")
			for i := 0; i < m.frequentProjectCount && i < len(m.projects); i++ {
				b.WriteString(m.renderProjectItem(i) + "\n")
			}
			if m.frequentProjectCount < len(m.projects) {
				b.WriteString("\n" + sectionStyle.Render("- Todos") + "\n")
			}
		} else {
			b.WriteString(sectionStyle.Render("- Todos") + "\n")
		}
		for i := m.frequentProjectCount; i < len(m.projects); i++ {
			b.WriteString(m.renderProjectItem(i) + "\n")
		}
	}

	b.WriteString(formFooter())
	return b.String()
}

func (m trackerModel) renderProjectItem(i int) string {
	p := m.projects[i]
	if i == m.formProjectCursor {
		return selectedStyle.Render(" > " + p.Name + " ")
	}
	return "   " + p.Name
}

func (m trackerModel) renderFormTags() string {
	var b strings.Builder
	b.WriteString(formHeader(4, 6, "Nueva Entrada"))
	b.WriteString("Selecciona las etiquetas (Espacio para marcar):\n\n")

	if len(m.tags) == 0 {
		b.WriteString(labelStyle.Render("No hay etiquetas disponibles.") + "\n")
	} else {
		if m.frequentTagCount > 0 {
			b.WriteString(sectionStyle.Render("- Frecuentes") + "\n")
			for i := 0; i < m.frequentTagCount && i < len(m.tags); i++ {
				b.WriteString(m.renderTagItem(i) + "\n")
			}
			if m.frequentTagCount < len(m.tags) {
				b.WriteString("\n" + sectionStyle.Render("- Todos") + "\n")
			}
		} else {
			b.WriteString(sectionStyle.Render("- Todos") + "\n")
		}
		for i := m.frequentTagCount; i < len(m.tags); i++ {
			b.WriteString(m.renderTagItem(i) + "\n")
		}
	}

	b.WriteString("\n" + helpStyle.Render("Espacio marcar/desmarcar  Enter continuar  Esc cancelar"))
	return b.String()
}

func (m trackerModel) renderTagItem(i int) string {
	t := m.tags[i]
	checked := "[ ]"
	checkStyle := labelStyle
	if m.formSelectedTags[t.ID] {
		checked = "[x]"
		checkStyle = tagCheckStyle
	}

	if i == m.formTagCursor {
		return selectedStyle.Render(" > " + checkStyle.Render(checked) + " " + t.Name)
	}
	return "   " + checkStyle.Render(checked) + " " + t.Name
}

func (m trackerModel) renderFormTime() string {
	var b strings.Builder
	b.WriteString(formHeader(5, 6, "Nueva Entrada"))
	b.WriteString("Tiempo en minutos:\n\n")
	b.WriteString(m.timeInput.View() + "\n")
	b.WriteString(labelStyle.Render("Ejemplos: 30, 60, 90, 480") + "\n")
	b.WriteString(formFooter())
	return b.String()
}

func (m trackerModel) renderFormBillable() string {
	var b strings.Builder
	b.WriteString(formHeader(6, 6, "Nueva Entrada"))
	b.WriteString("Es facturable?\n\n")

	noStyle := labelStyle
	yesStyle := labelStyle
	if m.formBillable {
		yesStyle = selectedStyle
	} else {
		noStyle = selectedStyle
	}

	b.WriteString(noStyle.Render("  NO  ") + "    " + yesStyle.Render("  SI  ") + "\n")
	b.WriteString("\n" + helpStyle.Render("<-- --> para cambiar  Enter guardar  Esc cancelar"))
	return b.String()
}
