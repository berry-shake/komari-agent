package server

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCommandOutputBounded(t *testing.T) {
	var out limitedOutput
	input := strings.Repeat("x", maxTaskOutput*4)
	n, err := out.Write([]byte(input))
	if err != nil || n != len(input) || len(out.String()) > maxTaskOutput+64 || !strings.Contains(out.String(), "truncated") {
		t.Fatal("output limit failed")
	}
}
func TestTaskTimeoutKillsSubprocess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix process-tree fixture")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	output, code := runTaskCommandContext(ctx, "sleep 30 & wait")
	if time.Since(started) > 3*time.Second || code == 0 || !strings.Contains(output, "deadline") {
		t.Fatalf("timeout failed: code=%d output=%q", code, output)
	}
}
func TestTaskConcurrencyIsBounded(t *testing.T) {
	slots := make(chan struct{}, 1)
	hold := make(chan struct{})
	done := make(chan struct{})
	if !launchBounded(slots, func() { <-hold; close(done) }) {
		t.Fatal("first task rejected")
	}
	if launchBounded(slots, func() { t.Error("overflow ran") }) {
		t.Fatal("limit bypass")
	}
	close(hold)
	<-done
}
