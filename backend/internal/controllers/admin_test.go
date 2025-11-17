package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	pdfimport "github.com/inscripcion-moodle/go-backend/internal/services/pdfimport"
)

type stubPDFImportService struct {
	result *pdfimport.PDFImportResult
	err    error
}

func (s *stubPDFImportService) ImportOfficialResultsPDF(_ context.Context, examID uint, _ string, replaceExisting bool) (*pdfimport.PDFImportResult, error) {
	if replaceExisting != s.result.ReplaceExisting {
		return nil, sqlErr{msg: "replace mismatch"}
	}
	s.result.ExamID = examID
	return s.result, s.err
}

type sqlErr struct {
	msg string
}

func (e sqlErr) Error() string { return e.msg }

func TestAdminImportEndpoint(t *testing.T) {
	handler := &AdminHandler{
		pdfImport: &stubPDFImportService{
			result: &pdfimport.PDFImportResult{
				ReplaceExisting: true,
				PageCount:       2,
				TextPreview:     "datos",
			},
		},
	}
	body, contentType := buildMultipartPDF(t)
	req := httptest.NewRequest(http.MethodPost, "/admin/exams/1/results/import?replace_existing=true", bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("exam_id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rr := httptest.NewRecorder()
	handler.importOfficialResults(rr, req)
	t.Logf("response: %s", rr.Body.String())
	require.Equal(t, http.StatusOK, rr.Code)
	var payload pdfimport.PDFImportResult
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&payload))
	require.Equal(t, uint(1), payload.ExamID)
	require.Equal(t, "datos", payload.TextPreview)
}

func buildMultipartPDF(t *testing.T) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "dummy.pdf")
	require.NoError(t, err)
	_, err = part.Write([]byte("%PDF-1.4"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return buf.Bytes(), writer.FormDataContentType()
}
