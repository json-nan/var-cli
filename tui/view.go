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
	flashStyle         = lipgloss.NewStyle().Foreground(colorSecondary).Bold(true)
	extraStyle         = lipgloss.NewStyle().Foreground(colorWarning).Bold(true)
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

func formatShortTime(minutes int) string {
	h := minutes / 60
	m := minutes % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%d:%02d", h, m)
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

var excludedGoalTagIDs = map[int]bool{
	3:   true, // Freelance
	104: true, // Overtime
	105: true, // Overtime
	106: true, // Overtime
}

func entryHasExcludedTag(e api.TimeEntry) bool {
	for _, t := range e.Tags {
		if excludedGoalTagIDs[t.ID] {
			return true
		}
	}
	return false
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

func todayGoalMinutes(entries []api.TimeEntry) int {
	today := time.Now().Format("2006-01-02")
	total := 0
	for _, e := range entries {
		if e.Date == today && !entryHasExcludedTag(e) {
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

func weekGoalMinutes(entries []api.TimeEntry) int {
	now := time.Now()
	weekStart := now.AddDate(0, 0, -int(now.Weekday())+1)
	if now.Weekday() == time.Sunday {
		weekStart = now.AddDate(0, 0, -6)
	}
	startStr := weekStart.Format("2006-01-02")
	total := 0
	for _, e := range entries {
		if e.Date >= startStr && !entryHasExcludedTag(e) {
			total += e.Minutes
		}
	}
	return total
}

func extraMinutes(entries []api.TimeEntry) int {
	total := 0
	for _, e := range entries {
		if entryHasExcludedTag(e) {
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

func (m trackerModel) renderFlash() string {
	if m.flash == "" {
		return ""
	}
	return flashStyle.Render(m.flash) + "\n\n"
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

	// Persistent summary (header + progress + week days)
	b.WriteString(m.renderPersistentSummary() + "\n")

	// Update banner
	if m.updateAvailable {
		b.WriteString(updateStyle.Render(fmt.Sprintf("v%s disponible -- presiona 'u' para actualizar", m.latestVersion)) + "\n")
	} else if m.updateError != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error al buscar actualizaciones: %v", m.updateError)) + "\n")
	}

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

	versionStr := m.currentVersion
	if versionStr != "" && versionStr != "dev" && !strings.HasPrefix(versionStr, "v") {
		versionStr = "v" + versionStr
	}
	versionRendered := mutedStyle.Render(versionStr)

	if len(parts) == 0 {
		return titleStyle.Render("VAR CLI") + "  " + versionRendered
	}
	return titleStyle.Render(strings.Join(parts, " / ")) + "  " + versionRendered
}

func (m trackerModel) renderProgressSection() string {
	var b strings.Builder

	todayMin := todayGoalMinutes(m.entries)
	weekMin := weekGoalMinutes(m.entries)
	// billMin := billableMinutes(m.entries)
	// totalMin := totalMinutes(m.entries)
	extraMin := extraMinutes(m.entries)

	// Day progress (per-day target based on weekday)
	now := time.Now()
	todayTarget := targetMinutesForWeekday(now.Weekday())
	if todayTarget == 0 {
		// weekend
		b.WriteString(progressLabelStyle.Render(fmt.Sprintf("Hoy:  %s (fin de semana)", formatMinutes(todayMin))) + "\n")
	} else {
		dayPct := float64(todayMin) / float64(todayTarget)
		if dayPct > 1 {
			dayPct = 1
		}
		dayBar := m.dayProgress.ViewAs(dayPct)
		dayLabel := fmt.Sprintf("Hoy:  %s / %s", formatMinutes(todayMin), formatMinutes(todayTarget))
		if todayMin > todayTarget {
			dayLabel = fmt.Sprintf("Hoy:  %s / %s  (%s extra)", formatMinutes(todayMin), formatMinutes(todayTarget), formatMinutes(todayMin-todayTarget))
		}
		b.WriteString(progressLabelStyle.Render(dayLabel) + "\n")
		b.WriteString(dayBar + "\n")
	}

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

	// Extra / Freelance / Overtime line
	if extraMin > 0 {
		b.WriteString(extraStyle.Render(fmt.Sprintf("Extra: %s", formatMinutes(extraMin))) + "\n")
	}

	// Billable summary
	// if billMin > 0 {
	// 	b.WriteString(billableStyle.Render(fmt.Sprintf("Facturable: %s  Total: %s", formatMinutes(billMin), formatMinutes(totalMin))) + "\n")
	// }

	return b.String()
}

func (m trackerModel) renderWeekDays() string {
	now := time.Now()
	weekStart := now.AddDate(0, 0, -int(now.Weekday())+1)
	if now.Weekday() == time.Sunday {
		weekStart = now.AddDate(0, 0, -6)
	}

	dayNames := []string{"Lun", "Mar", "Mie", "Jue", "Vie"}
	today := now.Format("2006-01-02")

	dateGoalMinutes := make(map[string]int)
	dateExtraMinutes := make(map[string]int)
	for _, e := range m.entries {
		if entryHasExcludedTag(e) {
			dateExtraMinutes[e.Date] += e.Minutes
		} else {
			dateGoalMinutes[e.Date] += e.Minutes
		}
	}

	var dayBoxes []string
	for i := 0; i < 5; i++ {
		date := weekStart.AddDate(0, 0, i).Format("2006-01-02")
		goalMin := dateGoalMinutes[date]
		extraMin := dateExtraMinutes[date]
		name := dayNames[i]
		isToday := date == today
		target := targetMinutesForWeekday(time.Weekday(i + 1))
		isFilled := goalMin >= target

		goalStr := "-"
		if goalMin > 0 {
			goalStr = formatShortTime(goalMin)
		}

		var extraStr string
		if extraMin > 0 {
			extraStr = "+" + formatShortTime(extraMin)
		}

		nameStyle := lipgloss.NewStyle().Width(7).Align(lipgloss.Center)
		goalStyle := lipgloss.NewStyle().Width(7).Align(lipgloss.Center)
		if isToday {
			nameStyle = nameStyle.Foreground(colorHighlight).Bold(true)
			goalStyle = goalStyle.Foreground(colorHighlight)
		} else if isFilled {
			nameStyle = nameStyle.Foreground(colorFg)
			goalStyle = goalStyle.Foreground(colorSecondary)
		} else if goalMin > 0 {
			nameStyle = nameStyle.Foreground(colorFg)
			goalStyle = goalStyle.Foreground(colorFg)
		} else {
			nameStyle = nameStyle.Foreground(colorMuted)
			goalStyle = goalStyle.Foreground(colorMuted)
		}

		nameLine := nameStyle.Render(name)
		goalLine := goalStyle.Render(goalStr)

		lines := []string{nameLine, goalLine}
		if extraStr != "" {
			extraLine := lipgloss.NewStyle().Width(7).Align(lipgloss.Center).Foreground(colorWarning).Render(extraStr)
			lines = append(lines, extraLine)
		}

		box := lipgloss.JoinVertical(lipgloss.Center, lines...)
		dayBoxes = append(dayBoxes, box)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, dayBoxes...)
}

func (m trackerModel) renderPersistentSummary() string {
	var b strings.Builder
	b.WriteString(m.renderHeader() + "\n")
	b.WriteString(m.renderProgressSection())
	b.WriteString(m.renderWeekDays() + "\n")
	return b.String()
}

func (m trackerModel) renderDateHeader(date string, entries []api.TimeEntry) string {
	goalTotal := 0
	extraTotal := 0
	for _, e := range entries {
		if entryHasExcludedTag(e) {
			extraTotal += e.Minutes
		} else {
			goalTotal += e.Minutes
		}
	}

	dn := dayName(date)
	today := time.Now().Format("2006-01-02")
	isToday := date == today

	text := fmt.Sprintf("%s  %s  (%s)", dn, date, formatMinutes(goalTotal))
	if extraTotal > 0 {
		text += extraStyle.Render(fmt.Sprintf(" +%s", formatMinutes(extraTotal)))
	}
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
	keys = append(keys, "e editar")
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
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))
	return dates
}

// ---- Form views ----

func formHeader(step, total int, title string) string {
	progress := fmt.Sprintf("Paso %d/%d", step, total)
	return titleStyle.Render(title) + "  " + labelStyle.Render(progress) + "\n\n"
}

func (m trackerModel) formTitle() string {
	if m.editingEntry != nil {
		return fmt.Sprintf("Editando Entrada #%d", m.editingEntry.ID)
	}
	return "Nueva Entrada"
}

func formFooter() string {
	return "\n" + helpStyle.Render("Enter continuar  Esc cancelar")
}

func (m trackerModel) renderFormDate() string {
	var b strings.Builder
	b.WriteString(m.renderPersistentSummary() + "\n")
	b.WriteString(m.renderFlash())
	b.WriteString(formHeader(1, 6, m.formTitle()))
	b.WriteString("Fecha (YYYY-MM-DD):\n\n")
	b.WriteString(m.dateInput.View() + "\n")

	emptyDays := m.emptyDays()
	if len(emptyDays) > 0 {
		b.WriteString("\n" + sectionStyle.Render("Dias sin completar:") + "\n")
		for i, d := range emptyDays {
			b.WriteString(m.renderEmptyDayItem(d, i) + "\n")
		}
		b.WriteString("\n" + helpStyle.Render("↑/↓ seleccionar dia  Enter continuar  Esc cancelar"))
	} else {
		b.WriteString(formFooter())
	}
	return b.String()
}

func (m trackerModel) renderEmptyDayItem(date string, idx int) string {
	t, _ := time.Parse("2006-01-02", date)
	dn := dayName(date)
	minutes := 0
	for _, e := range m.entries {
		if e.Date == date {
			minutes += e.Minutes
		}
	}
	target := targetMinutesForWeekday(t.Weekday())
	label := fmt.Sprintf("%s %s  (%s / %s)", dn, date, formatMinutes(minutes), formatMinutes(target))
	if idx == m.formEmptyDayCursor {
		return cursorStyle.Render(" > ") + labelStyle.Render(label)
	}
	return "   " + labelStyle.Render(label)
}

func (m trackerModel) renderFormDescription() string {
	var b strings.Builder
	b.WriteString(m.renderPersistentSummary() + "\n")
	b.WriteString(m.renderFlash())
	b.WriteString(formHeader(2, 6, m.formTitle()))
	b.WriteString("Descripcion:\n\n")
	b.WriteString(m.descInput.View() + "\n")

	suggestions := m.descSuggestions()
	if len(suggestions) > 0 {
		b.WriteString("\n" + sectionStyle.Render("Sugerencias:") + "\n")
		for i, e := range suggestions {
			b.WriteString(m.renderDescSuggestion(e, i) + "\n")
		}
		b.WriteString("\n" + helpStyle.Render("↑/↓ seleccionar  Enter autocompletar  Esc volver"))
	} else {
		b.WriteString("\n" + helpStyle.Render("Enter continuar  Esc volver"))
	}
	return b.String()
}

func (m trackerModel) renderDescSuggestion(e api.TimeEntry, idx int) string {
	desc := e.Description
	if len(desc) > 40 {
		desc = desc[:37] + "..."
	}
	parts := []string{
		fmt.Sprintf("[%s] %s", e.Project.Name, desc),
		formatMinutes(e.Minutes),
	}
	if e.IsBillable {
		parts = append(parts, billableStyle.Render("$"))
	}
	line := strings.Join(parts, "  ")
	if idx == m.formDescSuggestionCursor {
		return cursorStyle.Render(" > ") + line
	}
	return "   " + line
}

func (m trackerModel) renderFormProject() string {
	var b strings.Builder
	b.WriteString(m.renderPersistentSummary() + "\n")
	b.WriteString(m.renderFlash())
	b.WriteString(formHeader(3, 6, m.formTitle()))

	if len(m.projects) > 0 {
		selected := m.projects[m.formProjectCursor].Name
		b.WriteString(labelStyle.Render("Proyecto seleccionado: ") + selected + "\n\n")
	}
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

	b.WriteString("\n" + helpStyle.Render("↑/↓ navegar  Enter continuar  Esc volver"))
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
	b.WriteString(m.renderPersistentSummary() + "\n")
	b.WriteString(m.renderFlash())
	b.WriteString(formHeader(4, 6, m.formTitle()))

	var selectedTagNames []string
	for id := range m.formSelectedTags {
		for _, t := range m.tags {
			if t.ID == id {
				selectedTagNames = append(selectedTagNames, t.Name)
				break
			}
		}
	}
	if len(selectedTagNames) > 0 {
		b.WriteString(labelStyle.Render("Etiquetas seleccionadas: ") + strings.Join(selectedTagNames, ", ") + "\n\n")
	} else {
		b.WriteString(labelStyle.Render("Etiquetas seleccionadas: ") + "Ninguna" + "\n\n")
	}
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

	b.WriteString("\n" + helpStyle.Render("Espacio marcar/desmarcar  Enter continuar  Esc volver"))
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
	b.WriteString(m.renderPersistentSummary() + "\n")
	b.WriteString(m.renderFlash())
	b.WriteString(formHeader(5, 6, m.formTitle()))

	if val := m.timeInput.Value(); val != "" {
		if minutes, err := parseTimeInput(val); err == nil {
			b.WriteString(labelStyle.Render("Tiempo seleccionado: ") + formatMinutes(minutes) + "\n\n")
		}
	}
	b.WriteString("Duracion:\n\n")
	b.WriteString(m.timeInput.View() + "\n")
	b.WriteString(labelStyle.Render("Ejemplos: 30m, 1h, 1h30m, 5h, 5:30, 480") + "\n")
	b.WriteString("\n" + helpStyle.Render("Enter continuar  Esc volver"))
	return b.String()
}

func (m trackerModel) renderFormBillable() string {
	var b strings.Builder
	b.WriteString(m.renderPersistentSummary() + "\n")
	b.WriteString(m.renderFlash())
	b.WriteString(formHeader(6, 6, m.formTitle()))

	current := "NO"
	if m.formBillable {
		current = "SI"
	}
	b.WriteString(labelStyle.Render("Facturable: ") + current + "\n\n")
	b.WriteString("Es facturable?\n\n")

	noStyle := labelStyle
	yesStyle := labelStyle
	if m.formBillable {
		yesStyle = selectedStyle
	} else {
		noStyle = selectedStyle
	}

	b.WriteString(noStyle.Render("  NO  ") + "    " + yesStyle.Render("  SI  ") + "\n")
	b.WriteString("\n" + helpStyle.Render("<-- --> para cambiar  Enter guardar  Esc volver"))
	return b.String()
}
