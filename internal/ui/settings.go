package ui

import (
	"fmt"
)

type settingsItemType int

const (
	settingTypePath settingsItemType = iota
	settingTypeChatModel
	settingTypeDescModel
)

type settingsItem struct {
	title       string
	desc        string
	path        string
	itemType    settingsItemType
	currentVal  string
}

func newSettingsItem(title, description, path string, itemType settingsItemType, currentVal string) settingsItem {
	return settingsItem{
		title:      title,
		desc:       description,
		path:       path,
		itemType:   itemType,
		currentVal: currentVal,
	}
}

func (i settingsItem) Title() string       { return i.title }
func (i settingsItem) Description() string { return i.desc }
func (i settingsItem) FilterValue() string { return i.title + " " + i.desc }
func (i settingsItem) Path() string        { return i.path }

var knownModels = []string{
	"gemini-3-flash-preview",
	"gemini-2.5-pro-preview-03-25",
	"gemini-2.0-flash",
	"gemini-2.0-flash-lite",
	"gemini-1.5-pro",
	"gemini-1.5-flash",
}

func nextModel(current string) string {
	for i, m := range knownModels {
		if m == current && i+1 < len(knownModels) {
			return knownModels[i+1]
		}
	}
	return knownModels[0]
}

func buildSettingsItems(chatModel, descriptionModel, cfgPath, systemPromptPath, descriptionPromptPath string) []settingsItem {
	return []settingsItem{
		newSettingsItem("Chat Model", "Model used for conversations. Current: "+chatModel, "", settingTypeChatModel, chatModel),
		newSettingsItem("Description Model", "Model used for conversation titles. Current: "+descriptionModel, "", settingTypeDescModel, descriptionModel),
		newSettingsItem("Application Config", "Configure models, prompts, and paths. File: "+cfgPath, cfgPath, settingTypePath, ""),
		newSettingsItem("System Prompt", "Default assistant behavior and response policy. File: "+systemPromptPath, systemPromptPath, settingTypePath, ""),
		newSettingsItem("Description Prompt", "Conversation title generation prompt. File: "+descriptionPromptPath, descriptionPromptPath, settingTypePath, ""),
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
