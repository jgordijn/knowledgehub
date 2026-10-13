package routes

import (
	"net/http"
	"testing"

	"github.com/jgordijn/knowledgehub/internal/testutil"
)

func TestHandleMarkRead(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	res := testutil.CreateResource(t, app, "Blog", "https://example.com/feed", "rss", "healthy", 0, true)
	e1 := testutil.CreateEntry(t, app, res.Id, "One", "https://example.com/1", "g1")
	e2 := testutil.CreateEntry(t, app, res.Id, "Two", "https://example.com/2", "g2")
	e3 := testutil.CreateEntry(t, app, res.Id, "Three", "https://example.com/3", "g3")

	isRead := func(id string) bool {
		t.Helper()
		r, err := app.FindRecordById("entries", id)
		if err != nil {
			t.Fatalf("find %s: %v", id, err)
		}
		return r.GetBool("is_read")
	}

	status, result, err := HandleMarkRead(app, MarkReadInput{IDs: []string{e1.Id, e2.Id, "missing"}, IsRead: true})
	if err != nil || status != http.StatusOK {
		t.Fatalf("mark read: status=%d err=%v", status, err)
	}
	if result.Updated != 2 {
		t.Fatalf("expected 2 updated, got %d", result.Updated)
	}
	if !isRead(e1.Id) || !isRead(e2.Id) || isRead(e3.Id) {
		t.Fatalf("unexpected read state: e1=%v e2=%v e3=%v", isRead(e1.Id), isRead(e2.Id), isRead(e3.Id))
	}

	status, result, err = HandleMarkRead(app, MarkReadInput{IDs: []string{e1.Id}, IsRead: false})
	if err != nil || status != http.StatusOK || result.Updated != 1 {
		t.Fatalf("mark unread: status=%d updated=%d err=%v", status, result.Updated, err)
	}
	if isRead(e1.Id) || !isRead(e2.Id) {
		t.Fatalf("undo did not restore e1 only: e1=%v e2=%v", isRead(e1.Id), isRead(e2.Id))
	}
}

func TestHandleMarkReadEmptyAndTooMany(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	status, result, err := HandleMarkRead(app, MarkReadInput{IsRead: true})
	if err != nil || status != http.StatusOK || result.Updated != 0 {
		t.Fatalf("empty: status=%d updated=%d err=%v", status, result.Updated, err)
	}

	ids := make([]string, maxMarkReadIDs+1)
	for i := range ids {
		ids[i] = "x"
	}
	status, _, err = HandleMarkRead(app, MarkReadInput{IDs: ids, IsRead: true})
	if err == nil || status != http.StatusBadRequest {
		t.Fatalf("too many: status=%d err=%v", status, err)
	}
}
