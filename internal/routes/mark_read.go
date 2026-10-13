package routes

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/routine"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
	"github.com/pocketbase/pocketbase/tools/types"
)

// maxMarkReadIDs bounds a single bulk request; the feed loads at most 200 entries.
const maxMarkReadIDs = 1000

// MarkReadInput is the expected JSON body for the bulk mark-read endpoint.
type MarkReadInput struct {
	IDs    []string `json:"ids"`
	IsRead bool     `json:"is_read"`
}

// MarkReadResult is the JSON response from the bulk mark-read endpoint.
type MarkReadResult struct {
	Updated int64 `json:"updated"`
}

// RegisterMarkReadRoute adds POST /api/entries/mark-read.
func RegisterMarkReadRoute(se *core.ServeEvent) {
	se.Router.POST("/api/entries/mark-read", func(re *core.RequestEvent) error {
		if re.Auth == nil {
			return re.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required."})
		}
		var input MarkReadInput
		if err := re.BindBody(&input); err != nil {
			return re.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid mark-read payload."})
		}
		status, result, err := HandleMarkRead(re.App, input)
		if err != nil {
			return re.JSON(status, map[string]string{"error": err.Error()})
		}
		return re.JSON(status, result)
	})
}

// HandleMarkRead sets is_read for the given entries in a single UPDATE.
// It deliberately bypasses per-record saves: those load and re-serialize every
// entry (including raw_content) and broadcast a full realtime record per entry,
// which overwhelms the small host when marking a full page as read. Instead it
// broadcasts slim per-entry update events itself (see broadcastReadState).
func HandleMarkRead(app core.App, input MarkReadInput) (int, MarkReadResult, error) {
	if len(input.IDs) == 0 {
		return http.StatusOK, MarkReadResult{}, nil
	}
	if len(input.IDs) > maxMarkReadIDs {
		return http.StatusBadRequest, MarkReadResult{}, errors.New("Too many entries in one request.")
	}

	ids := make([]any, len(input.IDs))
	for i, id := range input.IDs {
		ids[i] = id
	}

	updated := types.NowDateTime().String()
	res, err := app.DB().Update("entries", dbx.Params{
		"is_read": input.IsRead,
		"updated": updated,
	}, dbx.In("id", ids...)).Execute()
	if err != nil {
		return http.StatusInternalServerError, MarkReadResult{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return http.StatusInternalServerError, MarkReadResult{}, err
	}

	var existing []string
	if err := app.DB().Select("id").From("entries").Where(dbx.In("id", ids...)).Column(&existing); err != nil {
		app.Logger().Warn("mark-read: could not load ids for realtime broadcast", "error", err)
	} else if err := broadcastReadState(app, existing, input.IsRead, updated); err != nil {
		app.Logger().Warn("mark-read: realtime broadcast failed", "error", err)
	}

	return http.StatusOK, MarkReadResult{Updated: n}, nil
}

// readStateRecord is the partial record sent in realtime events: only the
// fields mark-read changes, so subscribers merge it into what they have.
type readStateRecord struct {
	CollectionID   string `json:"collectionId"`
	CollectionName string `json:"collectionName"`
	ID             string `json:"id"`
	IsRead         bool   `json:"is_read"`
	Updated        string `json:"updated"`
}

type readStateEvent struct {
	Action string          `json:"action"`
	Record readStateRecord `json:"record"`
}

// broadcastReadState sends an "update" realtime event per entry to clients
// subscribed to entries, mirroring PocketBase's own topics and access checks.
func broadcastReadState(app core.App, ids []string, isRead bool, updated string) error {
	collection, err := app.FindCachedCollectionByNameOrId("entries")
	if err != nil {
		return err
	}

	for _, chunk := range app.SubscriptionsBroker().ChunkedClients(300) {
		for _, client := range chunk {
			var messages []subscriptions.Message
			clientAuth, _ := client.Get(apis.RealtimeClientAuthKey).(*core.Record)

			for _, id := range ids {
				prefixes := map[string]*string{
					collection.Name + "/" + id + "?": collection.ViewRule,
					collection.Id + "/" + id + "?":   collection.ViewRule,
					collection.Name + "/*?":          collection.ListRule,
					collection.Id + "/*?":            collection.ListRule,
				}
				for prefix, rule := range prefixes {
					for sub, options := range client.Subscriptions(prefix) {
						record := core.NewRecord(collection)
						record.Id = id
						requestInfo := &core.RequestInfo{
							Context: core.RequestInfoContextRealtime,
							Method:  http.MethodGet,
							Query:   options.Query,
							Headers: options.Headers,
							Auth:    clientAuth,
						}
						if ok, _ := app.CanAccessRecord(record, requestInfo, rule); !ok {
							continue
						}

						data, err := json.Marshal(readStateEvent{
							Action: "update",
							Record: readStateRecord{
								CollectionID:   collection.Id,
								CollectionName: collection.Name,
								ID:             id,
								IsRead:         isRead,
								Updated:        updated,
							},
						})
						if err != nil {
							return err
						}
						messages = append(messages, subscriptions.Message{Name: sub, Data: data})
					}
				}
			}

			if len(messages) > 0 {
				routine.FireAndForget(func() {
					for _, msg := range messages {
						client.Send(msg)
					}
				})
			}
		}
	}

	return nil
}
