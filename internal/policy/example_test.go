package policy

import "testing"

// README から案内するサンプルポリシーが、常に読み込めることを確かめる。
func TestExamplePolicyLoads(t *testing.T) {
	p, err := LoadFile("../../examples/basic/policy.cue")
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if p.RuleCount() != 8 {
		t.Errorf("RuleCount() = %d, want 8", p.RuleCount())
	}
}

func TestLoadFileMissing(t *testing.T) {
	if _, err := LoadFile("testdata/does-not-exist.cue"); err == nil {
		t.Error("LoadFile succeeded for a missing file")
	}
}
