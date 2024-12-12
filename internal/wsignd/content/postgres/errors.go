package postgres

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/wrale/wrale-signage/internal/wsignd/content"
	"github.com/wrale/wrale-signage/internal/wsignd/database"
)

// PostgreSQL error codes for JSONB operations
const (
	pgInvalidJSONBInput = "22P02" // invalid input syntax for type json
	pgInvalidJSONBPath  = "22030" // invalid input syntax for type jsonpath
	pgJSONBContainsNull = "22043" // null value cannot be processed in jsonb operation
	pgJSONBKeyTooLong   = "22026" // string too long for type character varying(n)
	pgJSONBParsingError = "22032" // invalid jsonb path expression
	pgJSONBTypeError    = "22033" // argument of jsonb operation is of wrong type
)

// mapPostgresError provides detailed error mapping for PostgreSQL errors,
// with special handling for JSONB operations
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
		return database.MapError(err, op)
	}

	// Map specific PostgreSQL errors
	switch pqErr.Code {
	case pgInvalidJSONBInput, pgInvalidJSONBPath, pgJSONBParsingError:
		return &content.Error{
			Code:    content.ErrCodeInvalidData,
			Message: fmt.Sprintf("invalid JSON data: %s", pqErr.Message),
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
	case pgJSONBKeyTooLong:
		return &content.Error{
			Code:    content.ErrCodeInvalidData,
			Message: "JSONB field value too long",
			Op:      op,
			Err:     err,
		}
	}

	// Handle foreign key violations
	if pqErr.Code == "23503" {
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
	if pqErr.Code == "23505" {
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
