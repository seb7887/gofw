package sietch

import (
	"testing"
)

func TestFilterBuilder(t *testing.T) {
	t.Run("NewFilter creates empty builder", func(t *testing.T) {
		builder := NewFilter()
		if builder == nil {
			t.Fatal("NewFilter() returned nil")
		}
		if len(builder.conditions) != 0 {
			t.Errorf("Expected empty conditions, got %d", len(builder.conditions))
		}
	})

	t.Run("Where adds condition", func(t *testing.T) {
		builder := NewFilter().Where("balance", OpGreaterThan, 100)
		if len(builder.conditions) != 1 {
			t.Errorf("Expected 1 condition, got %d", len(builder.conditions))
		}
		if builder.conditions[0].Field != "balance" {
			t.Errorf("Expected field 'balance', got '%s'", builder.conditions[0].Field)
		}
		if builder.conditions[0].Operator != OpGreaterThan {
			t.Errorf("Expected operator OpGreaterThan, got %v", builder.conditions[0].Operator)
		}
		if builder.conditions[0].Value != 100 {
			t.Errorf("Expected value 100, got %v", builder.conditions[0].Value)
		}
	})

	t.Run("Multiple Where calls chain correctly", func(t *testing.T) {
		builder := NewFilter().
			Where("balance", OpGreaterThan, 100).
			Where("status", OpEqual, "active")

		if len(builder.conditions) != 2 {
			t.Errorf("Expected 2 conditions, got %d", len(builder.conditions))
		}
	})

	t.Run("OrderBy adds sort field", func(t *testing.T) {
		builder := NewFilter().OrderBy("balance", SortDesc)
		if len(builder.sort) != 1 {
			t.Errorf("Expected 1 sort field, got %d", len(builder.sort))
		}
		if builder.sort[0].Field != "balance" {
			t.Errorf("Expected field 'balance', got '%s'", builder.sort[0].Field)
		}
		if builder.sort[0].Direction != SortDesc {
			t.Errorf("Expected direction SortDesc, got %v", builder.sort[0].Direction)
		}
	})

	t.Run("Multiple OrderBy calls chain correctly", func(t *testing.T) {
		builder := NewFilter().
			OrderBy("status", SortAsc).
			OrderBy("balance", SortDesc)

		if len(builder.sort) != 2 {
			t.Errorf("Expected 2 sort fields, got %d", len(builder.sort))
		}
	})

	t.Run("Limit sets limit", func(t *testing.T) {
		builder := NewFilter().Limit(10)
		if builder.limit == nil {
			t.Fatal("Limit is nil")
		}
		if *builder.limit != 10 {
			t.Errorf("Expected limit 10, got %d", *builder.limit)
		}
	})

	t.Run("Offset sets offset", func(t *testing.T) {
		builder := NewFilter().Offset(20)
		if builder.offset == nil {
			t.Fatal("Offset is nil")
		}
		if *builder.offset != 20 {
			t.Errorf("Expected offset 20, got %d", *builder.offset)
		}
	})

	t.Run("Distinct sets distinct flag", func(t *testing.T) {
		builder := NewFilter().Distinct()
		if !builder.distinct {
			t.Error("Expected distinct to be true")
		}
	})

	t.Run("Build creates Filter", func(t *testing.T) {
		filter := NewFilter().
			Where("balance", OpGreaterThan, 100).
			OrderBy("balance", SortDesc).
			Limit(10).
			Offset(20).
			Distinct().
			Build()

		if filter == nil {
			t.Fatal("Build() returned nil")
		}
		if len(filter.Conditions) != 1 {
			t.Errorf("Expected 1 condition, got %d", len(filter.Conditions))
		}
		if len(filter.Sort) != 1 {
			t.Errorf("Expected 1 sort field, got %d", len(filter.Sort))
		}
		if filter.Limit == nil || *filter.Limit != 10 {
			t.Error("Expected limit 10")
		}
		if filter.Offset == nil || *filter.Offset != 20 {
			t.Error("Expected offset 20")
		}
		if !filter.Distinct {
			t.Error("Expected distinct to be true")
		}
	})

	t.Run("Method chaining works", func(t *testing.T) {
		filter := NewFilter().
			Where("balance", OpGreaterThan, 100).
			Where("status", OpEqual, "active").
			OrderBy("created_at", SortDesc).
			Limit(50).
			Build()

		if filter == nil {
			t.Fatal("Build() returned nil")
		}
	})
}

func TestComparisonOperators(t *testing.T) {
	tests := []struct {
		name     string
		operator ComparisonOperator
		expected string
	}{
		{"OpEqual", OpEqual, "="},
		{"OpNotEqual", OpNotEqual, "!="},
		{"OpGreaterThan", OpGreaterThan, ">"},
		{"OpLessThan", OpLessThan, "<"},
		{"OpGreaterThanOrEqual", OpGreaterThanOrEqual, ">="},
		{"OpLessThanOrEqual", OpLessThanOrEqual, "<="},
		{"OpIn", OpIn, "IN"},
		{"OpNotIn", OpNotIn, "NOT IN"},
		{"OpLike", OpLike, "LIKE"},
		{"OpILike", OpILike, "ILIKE"},
		{"OpIsNull", OpIsNull, "IS NULL"},
		{"OpIsNotNull", OpIsNotNull, "IS NOT NULL"},
		{"OpBetween", OpBetween, "BETWEEN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.operator) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.operator))
			}
		})
	}
}

func TestSortDirection(t *testing.T) {
	t.Run("SortAsc", func(t *testing.T) {
		if string(SortAsc) != "ASC" {
			t.Errorf("Expected 'ASC', got '%s'", string(SortAsc))
		}
	})

	t.Run("SortDesc", func(t *testing.T) {
		if string(SortDesc) != "DESC" {
			t.Errorf("Expected 'DESC', got '%s'", string(SortDesc))
		}
	})
}

func TestCondition(t *testing.T) {
	t.Run("Condition struct creation", func(t *testing.T) {
		condition := Condition{
			Field:    "balance",
			Operator: OpEqual,
			Value:    100,
		}

		if condition.Field != "balance" {
			t.Errorf("Expected field 'balance', got '%s'", condition.Field)
		}
		if condition.Operator != OpEqual {
			t.Errorf("Expected operator OpEqual, got %v", condition.Operator)
		}
		if condition.Value != 100 {
			t.Errorf("Expected value 100, got %v", condition.Value)
		}
	})
}

func TestFilter(t *testing.T) {
	t.Run("Empty filter", func(t *testing.T) {
		filter := &Filter{}
		if len(filter.Conditions) != 0 {
			t.Error("Expected empty conditions")
		}
		if len(filter.Sort) != 0 {
			t.Error("Expected empty sort")
		}
		if filter.Limit != nil {
			t.Error("Expected nil limit")
		}
		if filter.Offset != nil {
			t.Error("Expected nil offset")
		}
		if filter.Distinct {
			t.Error("Expected distinct to be false")
		}
	})

	t.Run("Filter with all fields", func(t *testing.T) {
		limit := 10
		offset := 20
		filter := &Filter{
			Conditions: []Condition{
				{Field: "balance", Operator: OpGreaterThan, Value: 100},
			},
			Sort: []SortField{
				{Field: "balance", Direction: SortDesc},
			},
			Limit:    &limit,
			Offset:   &offset,
			Distinct: true,
		}

		if len(filter.Conditions) != 1 {
			t.Error("Expected 1 condition")
		}
		if len(filter.Sort) != 1 {
			t.Error("Expected 1 sort field")
		}
		if filter.Limit == nil || *filter.Limit != 10 {
			t.Error("Expected limit 10")
		}
		if filter.Offset == nil || *filter.Offset != 20 {
			t.Error("Expected offset 20")
		}
		if !filter.Distinct {
			t.Error("Expected distinct to be true")
		}
	})
}

func TestSortField(t *testing.T) {
	t.Run("SortField creation", func(t *testing.T) {
		sf := SortField{
			Field:     "balance",
			Direction: SortAsc,
		}

		if sf.Field != "balance" {
			t.Errorf("Expected field 'balance', got '%s'", sf.Field)
		}
		if sf.Direction != SortAsc {
			t.Errorf("Expected direction SortAsc, got %v", sf.Direction)
		}
	})
}
