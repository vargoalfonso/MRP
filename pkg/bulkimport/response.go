package bulkimport

import (
	"fmt"
	"net/http"

	"github.com/ganasa18/go-template/internal/base/app"
)

// HTTPResponse writes the appropriate JSON response for a bulk import result.
// baseURL is the scheme+host used to build the download_url (e.g. "http://localhost:8899").
func HTTPResponse(ctx *app.Context, result BulkResult, baseURL string) *app.CostumeResponse {
	type successData struct {
		ImportStatus ImportStatus `json:"import_status"`
		Total        int          `json:"total"`
		SuccessCount int          `json:"success_count"`
		FailedCount  int          `json:"failed_count"`
	}
	type errorData struct {
		ImportStatus ImportStatus `json:"import_status"`
		Total        int          `json:"total"`
		SuccessCount int          `json:"success_count"`
		FailedCount  int          `json:"failed_count"`
		DownloadURL  string       `json:"download_url,omitempty"`
	}

	switch result.Status {
	case StatusSuccess:
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusOK,
			Message:   "import berhasil",
			Data: successData{
				ImportStatus: StatusSuccess,
				Total:        result.Total,
				SuccessCount: result.SuccessCount,
				FailedCount:  0,
			},
		}
	case StatusPartial:
		dl := ""
		if result.ErrorToken != "" {
			dl = fmt.Sprintf("%s/api/v1/products/bom/import/errors/%s", baseURL, result.ErrorToken)
		}
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusMultiStatus,
			Message:   "import sebagian berhasil",
			Data: errorData{
				ImportStatus: StatusPartial,
				Total:        result.Total,
				SuccessCount: result.SuccessCount,
				FailedCount:  result.FailedCount,
				DownloadURL:  dl,
			},
		}
	default: // StatusFailed
		dl := ""
		if result.ErrorToken != "" {
			dl = fmt.Sprintf("%s/api/v1/products/bom/import/errors/%s", baseURL, result.ErrorToken)
		}
		return &app.CostumeResponse{
			RequestID: ctx.APIReqID,
			Status:    http.StatusUnprocessableEntity,
			Message:   "import gagal",
			Data: errorData{
				ImportStatus: StatusFailed,
				Total:        result.Total,
				SuccessCount: 0,
				FailedCount:  result.FailedCount,
				DownloadURL:  dl,
			},
		}
	}
}
