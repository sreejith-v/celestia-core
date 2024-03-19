package schema

import "github.com/tendermint/tendermint/pkg/trace"

const (
	// DumpTable is the name of the table that stores arbitrary data dumps.
	DumpTable = "dump"

	// DumpFieldKey is the name of the field that stores the arbitrary data.
	// The value is a string.
	DumpFieldKey = "key"

	// DumpValueFieldKey is the name of the field that stores the arbitrary data.
	// The value is a string. The encoding of that string is arbitrary.
	DumpValueFieldKey = "value"
)

// WriteDump writes a tracing point for a key-value pair using the predetermined
// schema for arbitrary data dumps. This is used to create a table in the following
// schema:
//
// | time | key | value |
func WriteDump(
	client *trace.Client,
	key, value string,
) {
	// this check is redundant to what is checked during WritePoint, although it
	// is an optimization to avoid allocations from the map of fields.
	if !client.IsCollecting(DumpTable) {
		return
	}
	client.WritePoint(DumpTable, map[string]interface{}{
		DumpFieldKey:      key,
		DumpValueFieldKey: value,
	})
}
