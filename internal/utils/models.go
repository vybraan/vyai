package utils


type Item struct {
	title string
	desc  string
}

func NewItem(title, description string) Item {
	return Item{
		title: title,
		desc:  description,
	}
}

func (i Item) Title() string       { return i.title }
func (i Item) Description() string { return i.desc }
func (i Item) FilterValue() string { return i.desc }
