package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"var-cli/api"
)

var (
	titleStyle      = lipgloss.NewStyle().MarginBottom(1).Foreground(lipgloss.Color("#FF7CCB")).Bold(true)
	subtitleStyle   = lipgloss.NewStyle().MarginBottom(1).Foreground(lipgloss.Color("#E0E0E0")).Bold(true)
	labelStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	entryStyle      = lipgloss.NewStyle().MarginLeft(2)
	dateHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	errorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F5F")).Bold(true)
	helpStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	billableStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	selectedStyle   = lipgloss.NewStyle().Background(lipgloss.Color("#7D56F4")).Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	cursorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
	sectionStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0")).Bold(true)
	tagCheckStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
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

func (m trackerModel) View() tea.View {
	var s string

	switch m.state {
	case stateInitializing:
		s = "Cargando configuración..."

	case stateVerifyingToken:
		s = titleStyle.Render("🔐 Autenticación") + "\n\n"
		s += m.loading

	case stateLogin:
		s = titleStyle.Render("🔐 Autenticación Requerida") + "\n\n"
		if m.err != nil {
			s += errorStyle.Render("Token inválido o expirado. Intenta de nuevo.") + "\n\n"
		}
		s += "Ingresa tu token de API para continuar:\n\n"
		s += m.tokenInput.View() + "\n\n"
		s += helpStyle.Render("Ctrl+V para pegar • Enter para guardar")

	case stateLoadingData:
		s = titleStyle.Render("📊 Cargando Datos") + "\n\n"
		s += m.loading

	case stateEntries:
		s = m.renderEntriesView()

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
		s = titleStyle.Render("💾 Guardando") + "\n\n"
		s += m.loading
	}

	finalView := lipgloss.NewStyle().Margin(1, 2).Render(s)
	return tea.NewView(finalView)
}

func (m trackerModel) renderEntriesView() string {
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("🕒 Entradas de Tiempo") + "\n")
	if m.profile != nil && m.profile.Name != "" {
		subtitle := fmt.Sprintf("👤 %s • %s", m.profile.Name, m.profile.Email)
		if m.profile.Position != "" {
			subtitle += fmt.Sprintf(" • %s", m.profile.Position)
		}
		b.WriteString(subtitleStyle.Render(subtitle) + "\n")
	} else if m.profile != nil {
		b.WriteString(errorStyle.Render("⚠️ Perfil recibido vacío") + "\n")
	} else {
		b.WriteString(errorStyle.Render("⚠️ Perfil no cargado") + "\n")
	}

	entries := m.displayEntries()
	if m.showAllEntries {
		b.WriteString(labelStyle.Render("Mostrando: últimas 2 semanas") + "\n\n")
	} else {
		b.WriteString(labelStyle.Render("Mostrando: últimos 7 días") + "\n\n")
	}

	// Weekly summary
	totalMin := totalMinutes(entries)
	totalHours := float64(totalMin) / 60.0
	b.WriteString(fmt.Sprintf("Total: %.1fh / 44h\n\n", totalHours))

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n")
	}

	// Entries grouped by date
	if len(entries) == 0 {
		b.WriteString("No hay entradas registradas en este período.\n")
	} else {
		entriesByDate := groupEntriesByDate(entries)
		for _, date := range sortedDates(entriesByDate) {
			b.WriteString(dateHeaderStyle.Render(date) + "\n")
			dayTotal := 0
			for _, e := range entriesByDate[date] {
				b.WriteString(m.renderEntry(e) + "\n")
				dayTotal += e.Minutes
			}
			b.WriteString(labelStyle.Render(fmt.Sprintf("  Total: %s", formatMinutes(dayTotal))) + "\n\n")
		}
	}

	toggle := "'a' ver todo"
	if m.showAllEntries {
		toggle = "'a' ver semana"
	}
	b.WriteString(helpStyle.Render("'n' nueva • 'r' recargar • " + toggle + " • 'q' salir"))
	return b.String()
}

func (m trackerModel) renderEntry(e api.TimeEntry) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("  • %s", e.Description))
	parts = append(parts, labelStyle.Render(fmt.Sprintf("[%s]", e.Project.Name)))
	parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("#F4D03F")).Render(formatMinutes(e.Minutes)))
	if e.IsBillable {
		parts = append(parts, billableStyle.Render("$"))
	}
	tagNames := m.getTagNames(e.Tags)
	if tagNames != "" {
		parts = append(parts, labelStyle.Render(fmt.Sprintf("(%s)", tagNames)))
	}
	return strings.Join(parts, " ")
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
	for i := 0; i < len(dates); i++ {
		for j := i + 1; j < len(dates); j++ {
			if dates[j] < dates[i] {
				dates[i], dates[j] = dates[j], dates[i]
			}
		}
	}
	return dates
}

// ── Form views ───────────────────────────────────────────────

func formHeader(step, total int, title string) string {
	progress := fmt.Sprintf("Paso %d/%d", step, total)
	return titleStyle.Render(title) + "  " + labelStyle.Render(progress) + "\n\n"
}

func formFooter() string {
	return "\n" + helpStyle.Render("Enter continuar • Esc cancelar")
}

func (m trackerModel) renderFormDate() string {
	var b strings.Builder
	b.WriteString(formHeader(1, 6, "📝 Nueva Entrada"))
	b.WriteString("Fecha (YYYY-MM-DD):\n\n")
	b.WriteString(m.dateInput.View() + "\n")
	b.WriteString(formFooter())
	return b.String()
}

func (m trackerModel) renderFormDescription() string {
	var b strings.Builder
	b.WriteString(formHeader(2, 6, "📝 Nueva Entrada"))
	b.WriteString("Descripción:\n\n")
	b.WriteString(m.descInput.View() + "\n")
	b.WriteString(formFooter())
	return b.String()
}

func (m trackerModel) renderFormProject() string {
	var b strings.Builder
	b.WriteString(formHeader(3, 6, "📝 Nueva Entrada"))
	b.WriteString("Selecciona el proyecto:\n\n")

	if len(m.projects) == 0 {
		b.WriteString(labelStyle.Render("No hay proyectos disponibles.") + "\n")
	} else {
		if m.frequentProjectCount > 0 {
			b.WriteString(sectionStyle.Render("▸ Frecuentes") + "\n")
			for i := 0; i < m.frequentProjectCount && i < len(m.projects); i++ {
				b.WriteString(m.renderProjectItem(i) + "\n")
			}
			if m.frequentProjectCount < len(m.projects) {
				b.WriteString("\n" + sectionStyle.Render("▸ Todos") + "\n")
			}
		} else {
			b.WriteString(sectionStyle.Render("▸ Todos") + "\n")
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
		return selectedStyle.Render(" › " + p.Name + " ")
	}
	return "   " + p.Name
}

func (m trackerModel) renderFormTags() string {
	var b strings.Builder
	b.WriteString(formHeader(4, 6, "📝 Nueva Entrada"))
	b.WriteString("Selecciona las etiquetas (Espacio para marcar):\n\n")

	if len(m.tags) == 0 {
		b.WriteString(labelStyle.Render("No hay etiquetas disponibles.") + "\n")
	} else {
		if m.frequentTagCount > 0 {
			b.WriteString(sectionStyle.Render("▸ Frecuentes") + "\n")
			for i := 0; i < m.frequentTagCount && i < len(m.tags); i++ {
				b.WriteString(m.renderTagItem(i) + "\n")
			}
			if m.frequentTagCount < len(m.tags) {
				b.WriteString("\n" + sectionStyle.Render("▸ Todos") + "\n")
			}
		} else {
			b.WriteString(sectionStyle.Render("▸ Todos") + "\n")
		}
		for i := m.frequentTagCount; i < len(m.tags); i++ {
			b.WriteString(m.renderTagItem(i) + "\n")
		}
	}

	b.WriteString("\n" + helpStyle.Render("Espacio marcar/desmarcar • Enter continuar • Esc cancelar"))
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
		return selectedStyle.Render(" › " + checkStyle.Render(checked) + " " + t.Name)
	}
	return "   " + checkStyle.Render(checked) + " " + t.Name
}

func (m trackerModel) renderFormTime() string {
	var b strings.Builder
	b.WriteString(formHeader(5, 6, "📝 Nueva Entrada"))
	b.WriteString("Tiempo en minutos:\n\n")
	b.WriteString(m.timeInput.View() + "\n")
	b.WriteString(labelStyle.Render("Ejemplos: 30, 60, 90, 480") + "\n")
	b.WriteString(formFooter())
	return b.String()
}

func (m trackerModel) renderFormBillable() string {
	var b strings.Builder
	b.WriteString(formHeader(6, 6, "📝 Nueva Entrada"))
	b.WriteString("¿Es facturable?\n\n")

	noStyle := labelStyle
	yesStyle := labelStyle
	if m.formBillable {
		yesStyle = selectedStyle
	} else {
		noStyle = selectedStyle
	}

	b.WriteString(noStyle.Render("  NO  ") + "    " + yesStyle.Render("  SÍ  ") + "\n")
	b.WriteString("\n" + helpStyle.Render("← → para cambiar • Enter guardar • Esc cancelar"))
	return b.String()
}
