package organize

// OrganizerFactory is a function that creates an Organizer
// This allows for dependency injection in tests
type OrganizerFactory func() Organizer

// Default factory that creates a real organizer
var DefaultOrganizerFactory OrganizerFactory = func() Organizer {
	return New()
}

// CurrentOrganizerFactory is the currently active factory
// This can be swapped in tests
var CurrentOrganizerFactory = DefaultOrganizerFactory

// SetOrganizerFactory sets a custom organizer factory for dependency injection
func SetOrganizerFactory(factory OrganizerFactory) {
	CurrentOrganizerFactory = factory
}

// ResetOrganizerFactory resets to the default organizer factory
func ResetOrganizerFactory() {
	CurrentOrganizerFactory = DefaultOrganizerFactory
}
