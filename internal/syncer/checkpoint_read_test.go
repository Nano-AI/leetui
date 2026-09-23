package syncer

import (
	"context"
	"strings"
	"testing"
)

func TestProblemsDoesNotIgnoreCheckpointReadFailure(t *testing.T) {
	st := testStore(t)
	if _, err := st.DB().Exec(`DROP TABLE sync_state`); err != nil {
		t.Fatal(err)
	}
	fake := &fakeLeetCode{total: 1}
	client, _ := newFake(t, fake)
	ch := make(chan Progress, 10)
	err := New(client, st, 100).Problems(context.Background(), ch, true)
	if err == nil || !strings.Contains(err.Error(), "read sync state") {
		t.Errorf("want checkpoint read failure, got %v", err)
	}
	updates := drainProgress(ch)
	if len(updates) == 0 || !updates[len(updates)-1].Finished || updates[len(updates)-1].Err == nil {
		t.Errorf("checkpoint failure did not reach progress: %+v", updates)
	}
	if fake.calls != 0 {
		t.Errorf("made %d requests despite unreadable resume state", fake.calls)
	}
}
