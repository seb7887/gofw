package sietch

import (
	"time"
)

// SoftDeletable is a marker interface for entities that support soft delete
// Entities implementing this interface will be soft-deleted instead of hard-deleted
type SoftDeletable interface {
	// IsDeleted returns true if the entity is marked as deleted
	IsDeleted() bool

	// SetDeleted marks the entity as deleted or undeleted
	SetDeleted(deleted bool)

	// GetDeletedAt returns the timestamp when the entity was deleted
	GetDeletedAt() *time.Time

	// SetDeletedAt sets the deletion timestamp
	SetDeletedAt(deletedAt *time.Time)
}

// SoftDeleteOptions configures soft delete behavior for a repository
type SoftDeleteOptions struct {
	// IncludeDeleted when true, queries will include soft-deleted records
	IncludeDeleted bool

	// DeletedAtField specifies the database column name for the deleted_at timestamp
	// Default: "deleted_at"
	DeletedAtField string

	// IsDeletedField specifies the database column name for the is_deleted flag
	// Default: "is_deleted"
	IsDeletedField string
}

// DefaultSoftDeleteOptions returns the default soft delete configuration
func DefaultSoftDeleteOptions() *SoftDeleteOptions {
	return &SoftDeleteOptions{
		IncludeDeleted: false,
		DeletedAtField: "deleted_at",
		IsDeletedField: "is_deleted",
	}
}

// isSoftDeletable checks if type T implements SoftDeletable interface
func isSoftDeletable[T any]() bool {
	var zero T
	_, ok := any(&zero).(SoftDeletable)
	return ok
}

// markAsDeleted marks an entity as soft-deleted
func markAsDeleted[T any](item *T) {
	if sd, ok := any(item).(SoftDeletable); ok {
		now := time.Now()
		sd.SetDeleted(true)
		sd.SetDeletedAt(&now)
	}
}

// isEntityDeleted checks if an entity is soft-deleted
func isEntityDeleted[T any](item *T) bool {
	if sd, ok := any(item).(SoftDeletable); ok {
		return sd.IsDeleted()
	}
	return false
}
