package postgres

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/wrale/wrale-signage/internal/wsignd/content"
	"github.com/wrale/wrale-signage/internal/wsignd/database"
)

// PostgreSQL error codes for common database errors
const (
	pgUniqueViolation     = "23505" // unique violation
	pgForeignKeyViolation = "23503" // foreign key violation
)

// PostgreSQL error codes for data type operations
const (
	pgInvalidInputSyntax = "22P02" // invalid input syntax for any type (json, array, etc.)
	pgInvalidJSONBPath   = "22030" // invalid input syntax for type jsonpath
	pgJSONBContainsNull  = "22043" // null value cannot be processed in jsonb operation
	pgJSONBTypeError     = "22033" // argument of jsonb operation is of wrong type
	pgArrayNullInput     = "2202E" // null value in array element
	pgArrayOutOfBounds   = "2202F" // array subscript error
	pgDivisionByZero     = "22012" // division by zero
)

// mapPostgresError provides detailed error mapping for PostgreSQL errors,
// with special handling for JSONB and array operations
func mapPostgresError(err error, op string) error {
	if err == nil {
		return nil
	}

	if err == sql.ErrNoRows {
		return content.ErrNotFound
	}

	// Cast to PostgreSQL error if possible
	pqErr, ok := err.(*pq.Error)
	if !ok {
		// Check for aggregation errors before falling back
		if isAggregationError(err) {
			return &content.Error{
				Code:    "CALCULATION_ERROR",
				Message: "failed to calculate metrics",
				Op:      op,
				Err:     err,
			}
		}
		return database.MapError(err, op)
	}

	// Map specific PostgreSQL errors
	switch pqErr.Code {
	case pgInvalidInputSyntax:
		// Check error message to determine specific type error
		msg := strings.ToLower(pqErr.Message)
		if strings.Contains(msg, "json") {
			return &content.Error{
				Code:    content.ErrCodeInvalidData,
				Message: fmt.Sprintf("invalid JSON data: %s", pqErr.Message),
				Op:      op,
				Err:     err,
			}
		}
		if strings.Contains(msg, "array") {
			return &content.Error{
				Code:    "CALCULATION_ERROR",
				Message: fmt.Sprintf("invalid array input: %s", pqErr.Message),
				Op:      op,
				Err:     err,
			}
		}
		// Generic invalid input error
		return &content.Error{
			Code:    content.ErrCodeInvalidData,
			Message: fmt.Sprintf("invalid input format: %s", pqErr.Message),
			Op:      op,
			Err:     err,
		}
	case pgInvalidJSONBPath:
		return &content.Error{
			Code:    content.ErrCodeInvalidData,
			Message: fmt.Sprintf("invalid JSONB path: %s", pqErr.Message),
			Op:      op,
			Err:     err,
		}
	case pgJSONBContainsNull, pgJSONBTypeError:
		return &content.Error{
			Code:    content.ErrCodeInvalidData,
			Message: fmt.Sprintf("JSONB type error: %s", pqErr.Message),
			Op:      op,
			Err:     err,
		}
	case pgArrayNullInput, pgArrayOutOfBounds:
		return &content.Error{
			Code:    "CALCULATION_ERROR",
			Message: fmt.Sprintf("array operation error: %s", pqErr.Message),
			Op:      op,
			Err:     err,
		}
	case pgDivisionByZero:
		return &content.Error{
			Code:    "CALCULATION_ERROR",
			Message: "division by zero in calculation",
			Op:      op,
			Err:     err,
		}
	}

	// Handle foreign key violations
	if pqErr.Code == pgForeignKeyViolation {
		// Extract the constraint name from the message
		parts := strings.Split(pqErr.Constraint, "_")
		if len(parts) > 0 {
			switch parts[len(parts)-1] {
			case "display", "displays":
				return content.ErrDisplayNotFound
			}
		}
		return &content.Error{
			Code:    content.ErrCodeInvalidReference,
			Message: fmt.Sprintf("invalid reference: %s", pqErr.Message),
			Op:      op,
			Err:     err,
		}
	}

	// Handle unique violations
	if pqErr.Code == pgUniqueViolation {
		return &content.Error{
			Code:    content.ErrCodeAlreadyExists,
			Message: fmt.Sprintf("resource already exists: %s", pqErr.Message),
			Op:      op,
			Err:     err,
		}
	}

	// Fall back to generic database error mapping
	return database.MapError(err, op)
}

// isAggregationError checks if error is related to array/aggregation operations
func isAggregationError(err error) bool {
	errStr := err.Error()
	return containsAny(errStr, []string{
		"array_agg",
		"aggregate",
		"division by zero",
		"null value",
		"invalid input syntax for type numeric",
		"cannot extract numeric value",
	})
}

// containsAny checks if str contains any of the substrings
func containsAny(str string, substrings []string) bool {
	for _, sub := range substrings {
		if strings.Contains(str, sub) {
			return true
		}
	}
	return false
}
