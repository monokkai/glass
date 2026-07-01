package stackparse

import "testing"

const sampleDump = `goroutine 1 [running]:
main.main()
	/tmp/demo/main.go:10 +0x1a

goroutine 5 [chan receive, 4 minutes]:
main.worker()
	/tmp/demo/main.go:20 +0x25
created by main.main
	/tmp/demo/main.go:15 +0x40

goroutine 6 [select (no cases), 3 minutes]:
main.main.func1()
	/tmp/demo/main.go:34 +0x18
created by main.main
	/tmp/demo/main.go:33 +0x50

goroutine 7 [select (no cases), 2 minutes]:
main.main.func1()
	/tmp/demo/main.go:34 +0x18
created by main.main
	/tmp/demo/main.go:33 +0x50`

func TestParse(t *testing.T) {
	got := Parse(sampleDump)
	if len(got) != 4 {
		t.Fatalf("expected 4 goroutines, got %d", len(got))
	}

	if got[0].ID != 1 || got[0].State != "running" || got[0].TopFrame != "main.main" {
		t.Errorf("goroutine 0 mismatch: %+v", got[0])
	}
	if got[1].WaitMinutes != 4 || got[1].State != "chan receive" {
		t.Errorf("goroutine 1 mismatch: %+v", got[1])
	}
	if got[2].TopFrame != "main.main.func1" || got[2].State != "select (no cases)" {
		t.Errorf("goroutine 2 mismatch: %+v", got[2])
	}
}

func TestGroup(t *testing.T) {
	gs := Parse(sampleDump)

	hp := Group(gs, nil)

	if len(hp) != 3 {
		t.Fatalf("expected 3 distinct frames, got %d: %+v", len(hp), hp)
	}

	if hp[0].Frame != "main.main.func1" || hp[0].Count != 2 {
		t.Errorf("expected main.main.func1 x2 first, got %+v", hp[0])
	}
	if !hp[0].Stuck {
		t.Errorf("expected main.main.func1 group to be flagged stuck (blocked >= 1min)")
	}
}

func TestGroupExternalStuckIDs(t *testing.T) {
	gs := Parse(sampleDump)
	hp := Group(gs, map[int]bool{1: true})

	var mainFrame *HotPath
	for i := range hp {
		if hp[i].Frame == "main.main" {
			mainFrame = &hp[i]
		}
	}
	if mainFrame == nil {
		t.Fatal("expected a main.main group")
	}
	if !mainFrame.Stuck {
		t.Errorf("expected main.main to be flagged stuck via external stuckIDs")
	}
}
