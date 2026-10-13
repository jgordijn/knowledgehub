package routes

import (
	"errors"
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
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
// entry (including raw_content) and broadcast a realtime event per record,
// which overwhelms the small host when marking a full page as read.
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

	res, err := app.DB().Update("entries", dbx.Params{
		"is_read": input.IsRead,
		"updated": types.NowDateTime().String(),
	}, dbx.In("id", ids...)).Execute()
	if err != nil {
		return http.StatusInternalServerError, MarkReadResult{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return http.StatusInternalServerError, MarkReadResult{}, err
	}
	return http.StatusOK, MarkReadResult{Updated: n}, nil
}
