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

func (m trackerModel) getProjectName(id int) string {
	for _, p := range m.projects {
		if p.ID == id {
			return p.Name
		}
	}
	return fmt.Sprintf("Proyecto %d", id)
}

func (m trackerModel) getTagNames(ids []int) string {
	var names []string
	for _, id := range ids {
		for _, t := range m.tags {
			if t.ID == id {
				names = append(names, t.Name)
				break
			}
		}
	}
	if len(names) == 0 {
		return ""
	}
	return strings.Join(names, ", ")
}

func (m trackerModel) weeklyTotal() int {
	total := 0
	for _, e := range m.entries {
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

	startDate, endDate := getWeekRange()
	b.WriteString(labelStyle.Render(fmt.Sprintf("Semana: %s → %s", startDate, endDate)) + "\n\n")

	// Weekly summary
	totalMinutes := m.weeklyTotal()
	totalHours := float64(totalMinutes) / 60.0
	b.WriteString(fmt.Sprintf("Total semanal: %.1fh / 44h\n\n", totalHours))

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n")
	}

	// Entries grouped by date
	if len(m.entries) == 0 {
		b.WriteString("No hay entradas registradas esta semana.\n")
	} else {
		entriesByDate := groupEntriesByDate(m.entries)
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

	b.WriteString(helpStyle.Render("Presiona 'q' para salir"))
	return b.String()
}

func (m trackerModel) renderEntry(e api.TimeEntry) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("  • %s", e.Description))
	parts = append(parts, labelStyle.Render(fmt.Sprintf("[%s]", m.getProjectName(e.ProjectID))))
	parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("#F4D03F")).Render(formatMinutes(e.Minutes)))
	if e.IsBillable {
		parts = append(parts, billableStyle.Render("$"))
	}
	tagNames := m.getTagNames(e.TagIDs)
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
	// Simple string sort works for YYYY-MM-DD format
	for i := 0; i < len(dates); i++ {
		for j := i + 1; j < len(dates); j++ {
			if dates[j] < dates[i] {
				dates[i], dates[j] = dates[j], dates[i]
			}
		}
	}
	return dates
}
