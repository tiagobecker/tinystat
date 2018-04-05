package client

// CreateAction exports the createAction method and
// calls it on the DefaultClient
func CreateAction(action string) error {
	return DefaultClient.createAction(action)
}

// GetActionCount exports the getActionCount method and
// calls it on the DefaultClient
func GetActionCount(action, duration string) (int64, error) {
	return DefaultClient.getActionCount(action, duration)
}
