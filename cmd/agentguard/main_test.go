package main

import "testing"

func TestRunRefusesToStart(t *testing.T) {
	if code := run(nil); code == 0 {
		t.Fatalf("run() = 0, want non-zero until the server is implemented")
	}
}
