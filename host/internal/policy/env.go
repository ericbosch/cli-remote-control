package policy

import (
	"sort"
	"strings"
)

// EngineEnv strips environment variables that are forbidden by policy from a base environment.
// It returns the sanitized env and a sorted list of removed keys (never includes values).
func EngineEnv(baseEnv []string) (sanitized []string, removedKeys []string) {
	removed := make(map[string]struct{})
	out := make([]string, 0, len(baseEnv))

	for _, kv := range baseEnv {
		k, _, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		// Policy: do not forward any *_API_KEY env vars into engine subprocesses.
		if strings.HasSuffix(k, "_API_KEY") {
			removed[k] = struct{}{}
			continue
		}
		out = append(out, kv)
	}

	keys := make([]string, 0, len(removed))
	for k := range removed {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return out, keys
}
