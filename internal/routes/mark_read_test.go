package routes

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/jgordijn/knowledgehub/internal/testutil"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
	"github.com/pocketbase/pocketbase/tools/types"
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

func TestHandleMarkReadBroadcastsSlimRealtimeUpdates(t *testing.T) {
	app, cleanup := testutil.NewTestApp(t)
	defer cleanup()

	user := testutil.CreateSuperuser(t, app, "realtime@example.com")
	res := testutil.CreateResource(t, app, "Blog", "https://example.com/feed", "rss", "healthy", 0, true)
	e1 := testutil.CreateEntry(t, app, res.Id, "One", "https://example.com/1", "g1")
	e2 := testutil.CreateEntry(t, app, res.Id, "Two", "https://example.com/2", "g2")

	// Use the production rules so unauthenticated subscribers are denied.
	entries, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		t.Fatalf("entries collection: %v", err)
	}
	entries.ListRule = types.Pointer("@request.auth.id != ''")
	entries.ViewRule = types.Pointer("@request.auth.id != ''")
	if err := app.Save(entries); err != nil {
		t.Fatalf("save entries rules: %v", err)
	}

	newClient :=func(auth *core.Record, subs ...string) *subscriptions.DefaultClient {
		c := subscriptions.NewDefaultClient()
		if auth != nil {
			c.Set(apis.RealtimeClientAuthKey, auth)
		}
		c.Subscribe(subs...)
		app.SubscriptionsBroker().Register(c)
		return c
	}
	wildcard := newClient(user, "entries/*")
	single := newClient(user, "entries/"+e2.Id)
	anonymous := newClient(nil, "entries/*")
	other := newClient(user, "resources/*")

	if _, _, err := HandleMarkRead(app, MarkReadInput{IDs: []string{e1.Id, e2.Id, "missing"}, IsRead: true}); err != nil {
		t.Fatalf("mark read: %v", err)
	}

	receive := func(c *subscriptions.DefaultClient, n int) []subscriptions.Message {
		t.Helper()
		var got []subscriptions.Message
		timeout := time.After(time.Second)
		for len(got) < n {
			select {
			case m := <-c.Channel():
				got = append(got, m)
			case <-timeout:
				t.Fatalf("expected %d messages, got %d", n, len(got))
			}
		}
		select {
		case m := <-c.Channel():
			t.Fatalf("unexpected extra message %s: %s", m.Name, m.Data)
		case <-time.After(50 * time.Millisecond):
		}
		return got
	}

	seen := map[string]bool{}
	for _, m := range receive(wildcard, 2) {
		if m.Name != "entries/*" {
			t.Fatalf("unexpected topic %q", m.Name)
		}
		var ev struct {
			Action string         `json:"action"`
			Record map[string]any `json:"record"`
		}
		if err := json.Unmarshal(m.Data, &ev); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if ev.Action != "update" || ev.Record["is_read"] != true || ev.Record["collectionName"] != "entries" {
			t.Fatalf("unexpected event: %s", m.Data)
		}
		if _, ok := ev.Record["raw_content"]; ok {
			t.Fatalf("event should not carry raw_content: %s", m.Data)
		}
		seen[ev.Record["id"].(string)] = true
	}
	if !seen[e1.Id] || !seen[e2.Id] {
		t.Fatalf("missing ids in events: %v", seen)
	}

	if got := receive(single, 1); got[0].Name != "entries/"+e2.Id {
		t.Fatalf("unexpected single-record topic %q", got[0].Name)
	}
	receive(anonymous, 0)
	receive(other, 0)
}
