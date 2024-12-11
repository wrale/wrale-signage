package rule

// options holds common flags used across rule commands
type options struct {
	// Core options
	priority    *int   // Rule evaluation priority
	siteID      string // Site ID selector
	zone        string // Zone selector
	position    string // Position selector
	contentType string // Content type to redirect to
	version     string // Content version
	hash        string // Content hash
	output      string // Output format for list command

	// Schedule options
	startTime  string   // Rule validity start time
	endTime    string   // Rule validity end time
	daysOfWeek []string // Active days of week
	timeOfDay  string   // Active time range within days

	// Order command options
	beforeRule  string // Place rule before this one
	afterRule   string // Place rule after this one
	moveToStart bool   // Move rule to start of list
	moveToEnd   bool   // Move rule to end of list
}
