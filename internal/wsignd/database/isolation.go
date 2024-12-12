package database

import "database/sql"

// Transaction isolation levels
const (
	// LevelDefault uses the database's default isolation level
	LevelDefault = sql.LevelDefault

	// LevelReadUncommitted provides no guarantees about isolation
	LevelReadUncommitted = sql.LevelReadUncommitted

	// LevelReadCommitted prevents dirty reads
	LevelReadCommitted = sql.LevelReadCommitted

	// LevelWriteCommitted prevents dirty reads and non-repeatable writes
	LevelWriteCommitted = sql.LevelWriteCommitted

	// LevelRepeatableRead prevents dirty reads and non-repeatable reads
	LevelRepeatableRead = sql.LevelRepeatableRead

	// LevelSerializable provides the highest isolation; prevents all anomalies
	LevelSerializable = sql.LevelSerializable

	// LevelLinearizable provides strict serializability (PostgreSQL only)
	LevelLinearizable = sql.LevelLinearizable

	// LevelSnapshot provides snapshot isolation (PostgreSQL only)
	LevelSnapshot = sql.LevelSnapshot
)
