package kv

const (
	// Inverted indexes: File<identifier>Idx constant used
	// as part of the filenames in the /snapshots/idx dir.
	//
	// Must have corresponding constants in tables.go files in the format:
	//
	// Tbl<identifier>Keys
	// Tbl<identifier>Idx
	//
	// They correspond to the "hot" DB tables for these indexes.
	FileLogAddressIdx = "logaddrs"
	FileLogTopicsIdx  = "logtopics"
	FileTracesFromIdx = "tracesfrom"
	FileTracesToIdx   = "tracesto"
)
