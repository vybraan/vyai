package utils

import "github.com/charmbracelet/bubbles/list"

func ConvertToItemList(items []Item) []list.Item {
	convertedItems := make([]list.Item, len(items))
	for i, item := range items {
		convertedItems[i] = item
	}
	return convertedItems
}
