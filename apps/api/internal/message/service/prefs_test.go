package service

import (
	"reflect"
	"testing"
)

func TestSplitMutedIgnoresRetiredWikiKeys(t *testing.T) {
	local, chatMuted := SplitMuted([]string{"liked", "wiki:banned", "chat"})
	if !chatMuted {
		t.Fatal("expected chat muted")
	}
	if !reflect.DeepEqual(local, []string{"liked"}) {
		t.Fatalf("local = %#v, want [liked]", local)
	}
}
