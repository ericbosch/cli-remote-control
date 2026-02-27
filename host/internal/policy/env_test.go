package policy

import (
	"reflect"
	"strings"
	"testing"
)

func TestEngineEnv_StripsAPIKeysOnly(t *testing.T) {
	in := []string{
		"PATH=/bin",
		"OPENAI_API_KEY=secret",
		"CURSOR_API_KEY=secret2",
		"MY_INTERNAL_API_KEY=secret3",
		"RC_TOKEN=should_stay",
		"TERM=xterm",
	}

	gotEnv, gotRemoved := EngineEnv(in)

	if want := []string{"CURSOR_API_KEY", "MY_INTERNAL_API_KEY", "OPENAI_API_KEY"}; !reflect.DeepEqual(gotRemoved, want) {
		t.Fatalf("removed keys mismatch: got=%v want=%v", gotRemoved, want)
	}

	for _, kv := range gotEnv {
		if strings.Contains(kv, "_API_KEY=") {
			t.Fatalf("sanitized env unexpectedly contains api key entry: %q", kv)
		}
	}

	// RC_TOKEN must be preserved.
	foundRCToken := false
	for _, kv := range gotEnv {
		if kv == "RC_TOKEN=should_stay" {
			foundRCToken = true
		}
	}
	if !foundRCToken {
		t.Fatalf("RC_TOKEN missing from sanitized env")
	}
}
