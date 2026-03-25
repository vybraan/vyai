package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/v2/list"
)

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

type conversationListItem struct {
	id    string
	title string
	desc  string
}

func newConversationListItem(id, title, model, updatedAt string) conversationListItem {
	description := fmt.Sprintf("%s  %s", model, updatedAt)
	return conversationListItem{
		id:    id,
		title: title,
		desc:  description,
	}
}

func (i conversationListItem) Title() string       { return i.title }
func (i conversationListItem) Description() string { return i.desc }
func (i conversationListItem) FilterValue() string { return i.title + " " + i.desc + " " + i.id }
func (i conversationListItem) ID() string          { return i.id }
