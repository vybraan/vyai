package ui

import "github.com/charmbracelet/bubbles/v2/list"

type settingsItem struct {
	title string
	desc  string
	path  string
}

func newSettingsItem(title, description, path string) settingsItem {
	return settingsItem{title: title, desc: description, path: path}
}

func (i settingsItem) Title() string       { return i.title }
func (i settingsItem) Description() string { return i.desc }
func (i settingsItem) FilterValue() string { return i.title + " " + i.desc }
func (i settingsItem) Path() string        { return i.path }

func buildSettingsItems(chatModel, descriptionModel, cfgPath, systemPromptPath, descriptionPromptPath string) []list.Item {
	return []list.Item{
		newSettingsItem("Application Config", "Models, paths, and data directory. Active chat model: "+chatModel, cfgPath),
		newSettingsItem("System Prompt", "Default assistant behavior and response policy", systemPromptPath),
		newSettingsItem("Description Prompt", "Conversation title generation prompt. Active model: "+descriptionModel, descriptionPromptPath),
	}
}
