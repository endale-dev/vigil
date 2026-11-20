package clock

import (
	"testing"
	"time"
)

func TestSystemNowUsesCurrentTime(t *testing.T) {
	before := time.Now()
	got := (System{}).Now()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Fatalf("System.Now() = %s, want a time between %s and %s", got, before, after)
	}
}
