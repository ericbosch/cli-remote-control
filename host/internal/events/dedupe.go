package events

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"sync"
)

type DedupeOptions struct {
	IncludeTimestampMS bool
}

type Deduper struct {
	mu   sync.Mutex
	max  int
	q    []string
	set  map[string]struct{}
	opts DedupeOptions
}

func NewDeduper(max int, opts DedupeOptions) *Deduper {
	if max <= 0 {
		max = 4096
	}
	return &Deduper{
		max:  max,
		q:    make([]string, 0, max),
		set:  make(map[string]struct{}, max),
		opts: opts,
	}
}

func (d *Deduper) Seen(ev SessionEvent) bool {
	key := DedupeKey(ev, d.opts)
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, ok := d.set[key]; ok {
		return true
	}
	d.set[key] = struct{}{}
	d.q = append(d.q, key)
	if len(d.q) > d.max {
		old := d.q[0]
		d.q = d.q[1:]
		delete(d.set, old)
	}
	return false
}

func DedupeKey(ev SessionEvent, opts DedupeOptions) string {
	payloadNorm := normalizePayload(ev.Payload)
	tsSuffix := ""
	if opts.IncludeTimestampMS {
		if ts, ok := extractTimestampMS(ev.Payload); ok {
			tsSuffix = "\n" + strconv.FormatInt(ts, 10)
		}
	}
	src := string(ev.Kind) + "\n" + ev.SessionID + "\n" + payloadNorm + tsSuffix
	sum := sha256.Sum256([]byte(src))
	return hex.EncodeToString(sum[:])
}

func normalizePayload(payload json.RawMessage) string {
	if len(payload) == 0 {
		return "null"
	}
	var v any
	if err := json.Unmarshal(payload, &v); err != nil {
		return string(payload)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return string(payload)
	}
	return string(b)
}

func extractTimestampMS(payload json.RawMessage) (int64, bool) {
	if len(payload) == 0 {
		return 0, false
	}
	var obj map[string]any
	if err := json.Unmarshal(payload, &obj); err != nil {
		return 0, false
	}
	for _, k := range []string{"timestamp_ms", "ts_ms"} {
		if raw, ok := obj[k]; ok {
			switch t := raw.(type) {
			case float64:
				return int64(t), true
			case int64:
				return t, true
			case json.Number:
				if n, err := t.Int64(); err == nil {
					return n, true
				}
			}
		}
	}
	return 0, false
}
