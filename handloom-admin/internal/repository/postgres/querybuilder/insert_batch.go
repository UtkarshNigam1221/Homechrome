package querybuilder

import (
	"fmt"
	"strings"
)

// BatchInsertBuilder constructs multi-row INSERT statements with automatic $N
// placeholder numbering.
type BatchInsertBuilder struct {
	table               string
	columns             []string
	rows                [][]interface{}
	onConflictDoNothing bool
}

// InsertBatch creates a new BatchInsertBuilder with fixed column names.
func InsertBatch(table string, columns ...string) *BatchInsertBuilder {
	return &BatchInsertBuilder{table: table, columns: columns}
}

// AddRow adds a row of values. The number of values must match the number of columns.
func (b *BatchInsertBuilder) AddRow(vals ...interface{}) *BatchInsertBuilder {
	b.rows = append(b.rows, vals)
	return b
}

// OnConflictDoNothing appends ON CONFLICT DO NOTHING to the INSERT.
func (b *BatchInsertBuilder) OnConflictDoNothing() *BatchInsertBuilder {
	b.onConflictDoNothing = true
	return b
}

// Build assembles the final INSERT SQL and argument slice.
// Returns empty string and nil args if no rows were added.
func (b *BatchInsertBuilder) Build() (string, []interface{}) {
	if len(b.rows) == 0 {
		return "", nil
	}

	colCount := len(b.columns)
	var sb strings.Builder
	args := make([]interface{}, 0, len(b.rows)*colCount)

	sb.WriteString("INSERT INTO ")
	sb.WriteString(b.table)
	sb.WriteString(" (")
	sb.WriteString(strings.Join(b.columns, ", "))
	sb.WriteString(") VALUES ")

	for i, row := range b.rows {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(")
		for j := range row {
			if j > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("$%d", i*colCount+j+1))
		}
		sb.WriteString(")")
		args = append(args, row...)
	}

	if b.onConflictDoNothing {
		sb.WriteString(" ON CONFLICT DO NOTHING")
	}

	return sb.String(), args
}
