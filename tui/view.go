package tui

import (
	"fmt"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	weeklyHours = 44
	titleStyle = lipgloss.NewStyle().MarginBottom(1).Foreground(lipgloss.Color("#FF7CCB")).Bold(true)
)

func (m trackerModel) View() tea.View {
	var s string

	switch m.state {
	case stateInitializing:
		s = "Cargando configuración..."

	case stateLogin:
		s = titleStyle.Render("🔐 Autenticación Requerida") + "\n\n"
		s += "Ingresa tu token de API para continuar:\n\n"
		s += m.tokenInput.View() + "\n\n"
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("(Presiona Enter para guardar)")

	case stateMain:
		s = titleStyle.Render("🕒 Tracker de Horas Semanales") + "\n"
		s += fmt.Sprintf("Horas registradas: %.0f / %d\n\n", m.currentHours, weeklyHours)

		porcentaje := m.currentHours / float64(weeklyHours)
		s += m.progressBar.ViewAs(porcentaje) + "\n\n"

		s += lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("Presiona 'a' para agregar hora • 'q' para salir")
	}

	finalView := lipgloss.NewStyle().Margin(1, 2).Render(s)
	return tea.NewView(finalView)
}
