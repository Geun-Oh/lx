// Package buffer provides memory-managed buffering for the log pipeline.
package buffer

import (
	"sync"

	"github.com/Geun-Oh/lx/internal/entry"
)

// Pool manages a pool of reusable LogEntry objects to reduce GC pressure.
var Pool = &sync.Pool{
	New: func() interface{} {
		return &entry.LogEntry{
			Fields: make(map[string]string, 4),
		}
	},
}

// Get retrieves a LogEntry from the pool with fields map pre-allocated.
func Get() *entry.LogEntry {
	e := Pool.Get().(*entry.LogEntry)
	// Reset fields.
	for k := range e.Fields {
		delete(e.Fields, k)
	}
	e.Message = ""
	e.Raw = e.Raw[:0]
	e.Stream = ""
	e.Source = ""
	e.Level = entry.LevelUnknown
	e.Seq = 0
	return e
}

// Put returns a LogEntry to the pool for reuse.
func Put(e *entry.LogEntry) {
	Pool.Put(e)
}
