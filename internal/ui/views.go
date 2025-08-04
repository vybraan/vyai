package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss/v2"
)

func (m UIModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	//Tabirization
	switch m.activeTab {
	case 0:
		if m.loading {
			// Testing to show spinner when waiting for the models response
			loadGap := "\n\n"
			return fmt.Sprintf(
				"%s\n%s%s%s\n%s", m.headerView(),
				m.viewport.View(),
				loadGap,
				m.spinner.View()+" Thinking...\n", m.footerView(),
			)
		}

		return fmt.Sprintf(
			"%s\n%s%s%s\n%s", m.headerView(),
			m.viewport.View(),
			gap,
			m.textarea.View(), m.footerView(),
		)
	case 1:
		return fmt.Sprintf("%s\n%s", m.headerView(), m.theme.DocStyle.Render(m.explore.View()))
	default:
		return fmt.Sprintf("%s\n%s", m.headerView(), "Screwed up ailton \n * "+m.Tabs[m.activeTab]+" Page under development")

	}
}

func (m UIModel) headerView() string {

	// Status bar
	w := lipgloss.Width
	statusKey := m.theme.TopStatusBar.Render("VYAI")

	//---Tabs
	var renderedTabs []string
	for i, tab := range m.Tabs {
		if m.activeTab == i {

			if m.state == Insert {
				renderedTabs = append(renderedTabs, m.theme.ModeInsert.Render(tab+" (active)"))
			} else {

				renderedTabs = append(renderedTabs, m.theme.ModeNormal.Render(tab+" (active)"))
			}
			continue
		}
		if m.state == Insert {
			renderedTabs = append(renderedTabs, m.theme.BaseInsert.Render(tab))
		} else {
			renderedTabs = append(renderedTabs, m.theme.BaseNormal.Render(tab))
		}
	}
	tabs := lipgloss.JoinHorizontal(lipgloss.Center, renderedTabs...)

	placeholderWidth := 0
	for _, tab := range renderedTabs {
		placeholderWidth += w(tab)
	}
	placeholder := m.theme.StatusBar.Width(m.viewport.Width() - w(statusKey) - placeholderWidth).Render("")

	bar := lipgloss.JoinHorizontal(lipgloss.Top,
		statusKey,
		tabs,
		placeholder,
	)

	return bar
}

func (m UIModel) footerView() string {

	// Status bar
	w := lipgloss.Width

	var viewPortPercent, encoding, modelKey, status string

	if m.state == Insert {
		status = m.theme.ModeInsert.Render(string(m.state))
		encoding = m.theme.BaseInsert.Render("UTF-8")
		modelKey = m.theme.BaseInsert.Render("Model")

		viewPortPercent = m.theme.ModeInsert.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	} else {
		status = m.theme.ModeNormal.Render(string(m.state))
		encoding = m.theme.BaseNormal.Render("UTF-8")
		modelKey = m.theme.BaseNormal.Render("Model")

		viewPortPercent = m.theme.ModeNormal.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	}

	modelVal := m.theme.BottomModelTxt.
		Width(m.viewport.Width() - w(modelKey) - w(status) - w(encoding) - w(viewPortPercent)).
		Render("gemini-2.0-flash")

	bar := lipgloss.JoinHorizontal(lipgloss.Top,
		status,
		modelKey,
		modelVal,
		encoding,
		viewPortPercent,
	)

	return bar
}
