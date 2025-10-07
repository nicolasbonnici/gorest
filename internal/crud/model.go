package crud

// Model is the interface that all generated models must implement
type Model interface {
	TableName() string
}
