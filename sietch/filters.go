package sietch

// ComparisonOperator represents a type-safe comparison operator
type ComparisonOperator string

const (
	// Basic comparison operators
	OpEqual              ComparisonOperator = "="
	OpNotEqual           ComparisonOperator = "!="
	OpGreaterThan        ComparisonOperator = ">"
	OpLessThan           ComparisonOperator = "<"
	OpGreaterThanOrEqual ComparisonOperator = ">="
	OpLessThanOrEqual    ComparisonOperator = "<="

	// Advanced operators
	OpIn        ComparisonOperator = "IN"        // Value should be a slice
	OpNotIn     ComparisonOperator = "NOT IN"    // Value should be a slice
	OpLike      ComparisonOperator = "LIKE"      // Pattern matching (case-sensitive)
	OpILike     ComparisonOperator = "ILIKE"     // Pattern matching (case-insensitive)
	OpIsNull    ComparisonOperator = "IS NULL"   // Value is ignored
	OpIsNotNull ComparisonOperator = "IS NOT NULL" // Value is ignored
	OpBetween   ComparisonOperator = "BETWEEN"   // Value should be [2]any{min, max}
)

// SortDirection represents the sorting direction
type SortDirection string

const (
	SortAsc  SortDirection = "ASC"
	SortDesc SortDirection = "DESC"
)

// SortField represents a field to sort by with its direction
type SortField struct {
	Field     string
	Direction SortDirection
}

// LogicalOperator represents logical operators for combining conditions
type LogicalOperator string

const (
	LogicalAND LogicalOperator = "AND"
	LogicalOR  LogicalOperator = "OR"
	LogicalNOT LogicalOperator = "NOT"
)

// Condition represents a condition to filter queries.
// It can be either a leaf condition (field comparison) or a composite condition (logical grouping)
type Condition struct {
	// Leaf condition fields (for simple comparisons)
	Field    string
	Operator ComparisonOperator
	Value    any

	// Composite condition fields (for logical grouping)
	LogicalOp  LogicalOperator // AND, OR, NOT
	Conditions []Condition     // Nested conditions for composite
}

// IsLeaf returns true if this is a leaf condition (field comparison)
func (c *Condition) IsLeaf() bool {
	return c.LogicalOp == "" && len(c.Conditions) == 0
}

// IsComposite returns true if this is a composite condition (logical grouping)
func (c *Condition) IsComposite() bool {
	return c.LogicalOp != "" && len(c.Conditions) > 0
}

// Filter groups a set of conditions with sorting, pagination, and distinct options
type Filter struct {
	Conditions []Condition
	Sort       []SortField // Multiple fields for composite sorting
	Limit      *int        // Pointer to distinguish between 0 and not set
	Offset     *int        // For pagination
	Distinct   bool        // Return distinct results
}

// FilterBuilder provides a fluent interface for building filters
type FilterBuilder struct {
	conditions []Condition
	sort       []SortField
	limit      *int
	offset     *int
	distinct   bool
}

// NewFilter creates a new FilterBuilder
func NewFilter() *FilterBuilder {
	return &FilterBuilder{
		conditions: make([]Condition, 0),
		sort:       make([]SortField, 0),
	}
}

// Where adds a condition to the filter
func (fb *FilterBuilder) Where(field string, op ComparisonOperator, value any) *FilterBuilder {
	fb.conditions = append(fb.conditions, Condition{
		Field:    field,
		Operator: op,
		Value:    value,
	})
	return fb
}

// Or adds an OR condition grouping multiple conditions
// All conditions within the OR group will be combined with OR logic
func (fb *FilterBuilder) Or(conditions ...Condition) *FilterBuilder {
	if len(conditions) == 0 {
		return fb
	}
	fb.conditions = append(fb.conditions, Condition{
		LogicalOp:  LogicalOR,
		Conditions: conditions,
	})
	return fb
}

// And adds an AND condition grouping multiple conditions
// All conditions within the AND group will be combined with AND logic
func (fb *FilterBuilder) And(conditions ...Condition) *FilterBuilder {
	if len(conditions) == 0 {
		return fb
	}
	fb.conditions = append(fb.conditions, Condition{
		LogicalOp:  LogicalAND,
		Conditions: conditions,
	})
	return fb
}

// Not negates a condition or group of conditions
func (fb *FilterBuilder) Not(condition Condition) *FilterBuilder {
	fb.conditions = append(fb.conditions, Condition{
		LogicalOp:  LogicalNOT,
		Conditions: []Condition{condition},
	})
	return fb
}

// Group creates a logical grouping of conditions with the specified operator
func (fb *FilterBuilder) Group(op LogicalOperator, conditions ...Condition) *FilterBuilder {
	if len(conditions) == 0 {
		return fb
	}
	fb.conditions = append(fb.conditions, Condition{
		LogicalOp:  op,
		Conditions: conditions,
	})
	return fb
}

// OrderBy adds a sort field to the filter
func (fb *FilterBuilder) OrderBy(field string, direction SortDirection) *FilterBuilder {
	fb.sort = append(fb.sort, SortField{
		Field:     field,
		Direction: direction,
	})
	return fb
}

// Limit sets the maximum number of results to return
func (fb *FilterBuilder) Limit(n int) *FilterBuilder {
	fb.limit = &n
	return fb
}

// Offset sets the number of results to skip
func (fb *FilterBuilder) Offset(n int) *FilterBuilder {
	fb.offset = &n
	return fb
}

// Distinct sets the distinct flag
func (fb *FilterBuilder) Distinct() *FilterBuilder {
	fb.distinct = true
	return fb
}

// Build creates the final Filter
func (fb *FilterBuilder) Build() *Filter {
	return &Filter{
		Conditions: fb.conditions,
		Sort:       fb.sort,
		Limit:      fb.limit,
		Offset:     fb.offset,
		Distinct:   fb.distinct,
	}
}
