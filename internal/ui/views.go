package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss/v2"
)

func (m UIModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	notice := m.noticeView()

	switch m.activeTab {
	case 0:
		return fmt.Sprintf(
			"%s%s\n%s%s%s\n%s", m.headerView(),
			notice,
			m.viewport.View(),
			gap,
			m.inputView(),
			m.footerView(),
		)
	case 1:
		return fmt.Sprintf("%s%s\n%s", m.headerView(), notice, m.theme.DocStyle.Render(m.explore.View()))
	case 2:
		return fmt.Sprintf("%s%s\n%s", m.headerView(), notice, m.settingsView())
	default:
		return fmt.Sprintf("%s%s\n%s", m.headerView(), notice, "Unknown tab")

	}
}

func (m UIModel) settingsView() string {
	var items []string
	for i, item := range m.settingsItems {
		title := item.title
		desc := item.desc
		if i == m.settingsIndex {
			title = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7aa2f7")).
				Bold(true).
				Render("▸ " + title)
			desc = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7aa2f7")).
				Render(desc)
		} else {
			title = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#D9DCCF")).
				Render("  " + title)
			desc = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#777777")).
				Render("  " + desc)
		}
		block := lipgloss.JoinVertical(lipgloss.Left, title, desc)
		if i < len(m.settingsItems)-1 {
			block += "\n"
		}
		items = append(items, block)
	}
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#858392")).
		Render("Settings  Enter: toggle/cycle value")
	return header + "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)
}

func (m UIModel) inputView() string {
	if !m.loading || m.streaming {
		return m.textarea.View()
	}
	return lipgloss.NewStyle().Height(lipgloss.Height(m.textarea.View())).Render(m.spinner.View() + " Thinking...")
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
	placeholder := m.theme.StatusBar.Width(max(0, m.width-w(statusKey)-placeholderWidth)).Render("")

	bar := lipgloss.JoinHorizontal(lipgloss.Top,
		statusKey,
		tabs,
		placeholder,
	)

	return lipgloss.NewStyle().MaxWidth(m.width).MaxHeight(1).Render(bar)
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
		Width(max(1, m.width-w(modelKey)-w(status)-w(encoding)-w(viewPortPercent))).
		MaxWidth(max(1, m.width-w(modelKey)-w(status)-w(encoding)-w(viewPortPercent))).
		MaxHeight(1).
		Render(m.gsService.Config().ChatModel)

	bar := lipgloss.JoinHorizontal(lipgloss.Top,
		status,
		modelKey,
		modelVal,
		encoding,
		viewPortPercent,
	)

	return lipgloss.NewStyle().MaxWidth(m.width).MaxHeight(1).Render(bar)
}

func (m UIModel) noticeView() string {
	if m.notice == "" {
		return ""
	}

	return "\n" + lipgloss.NewStyle().
		Width(max(1, m.width)).
		Foreground(lipgloss.Color("#F5E6A7")).
		Background(lipgloss.Color("#5C3B00")).
		Padding(0, 1).
		Render(m.notice)
}
