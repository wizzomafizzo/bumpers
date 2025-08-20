package settings

// Editor provides methods for editing Claude settings files.
type Editor struct{}

// NewEditor creates a new settings editor.
func NewEditor() *Editor {
	return &Editor{}
}

// Load loads settings from a file.
func (*Editor) Load(filename string) (*Settings, error) {
	return LoadFromFile(filename)
}

// Save saves settings to a file.
func (*Editor) Save(settings *Settings, filename string) error {
	return SaveToFile(settings, filename)
}
