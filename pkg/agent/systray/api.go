package systray

type Menu struct {
	Icon    string     `json:"icon"`
	Title   string     `json:"title"`
	Tooltip string     `json:"tooltip"`
	Items   []MenuItem `json:"items"`
}

type MenuItem struct {
	Title   string `json:"title"`
	Icon    string `json:"icon"`
	Tooltip string `json:"tooltip"`
	Enabled bool   `json:"enabled"`
	Checked bool   `json:"checked"`
}

type Message struct {
	Type  MessageType `json:"type"`
	Item  *MenuItem   `json:"item,omitempty"`
	Menu  *Menu       `json:"menu,omitempty"`
	Error *string     `json:"error,omitempty"`
}

type MessageType string

const (
	InitMenu    MessageType = "init-menu"
	ItemClicked MessageType = "item-clicked"
	Error       MessageType = "error"
)
