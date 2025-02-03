package sietch

// Condition represents a condition to filter queries
type Condition struct {
	Field    string
	Operator string
	Value    any
}

// Filter groups a set of conditions
type Filter struct {
	Conditions []Condition
}
