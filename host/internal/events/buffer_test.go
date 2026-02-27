package events

import "testing"

func TestBufferBoundsAndReplay(t *testing.T) {
	b := NewBuffer(3)
	for i := 0; i < 5; i++ {
		b.Append(SessionEvent{SessionID: "s1", Engine: "shell", Kind: EventKindStatus})
	}
	if got := b.LastSeq(); got != 5 {
		t.Fatalf("LastSeq=%d want=5", got)
	}

	all := b.ReplayFromSeq(0)
	if len(all) != 3 {
		t.Fatalf("ReplayFromSeq(0) len=%d want=3", len(all))
	}
	if all[0].Seq != 3 || all[1].Seq != 4 || all[2].Seq != 5 {
		t.Fatalf("seqs=%d,%d,%d want=3,4,5", all[0].Seq, all[1].Seq, all[2].Seq)
	}

	from3 := b.ReplayFromSeq(3)
	if len(from3) != 2 || from3[0].Seq != 4 || from3[1].Seq != 5 {
		t.Fatalf("ReplayFromSeq(3) unexpected: %+v", from3)
	}

	last2 := b.ReplayLastN(2)
	if len(last2) != 2 || last2[0].Seq != 4 || last2[1].Seq != 5 {
		t.Fatalf("ReplayLastN(2) unexpected: %+v", last2)
	}
}
