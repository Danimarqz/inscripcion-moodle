package controllers

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

	"github.com/inscripcion-moodle/go-backend/internal/models"
)

type stubExcelImportService struct {
	result *models.ExcelImportResult
	err    error
}

func (s *stubExcelImportService) ImportOfficialResultsExcel(_ context.Context, examID uint, _ string, replaceExisting bool) (*models.ExcelImportResult, error) {
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
	Controller := &AdminController{
		excelImport: &stubExcelImportService{
			result: &models.ExcelImportResult{
				ReplaceExisting: true,
			},
		},
	}
	body, contentType := buildMultipartExcel(t)
	req := httptest.NewRequest(http.MethodPost, "/admin/exams/1/results/import?replace_existing=true", bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("exam_id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rr := httptest.NewRecorder()
	Controller.importOfficialResults(rr, req)
	t.Logf("response: %s", rr.Body.String())
	require.Equal(t, http.StatusOK, rr.Code)
	var payload models.ExcelImportResult
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&payload))
	require.Equal(t, uint(1), payload.ExamID)
}

func buildMultipartExcel(t *testing.T) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "dummy.xlsx")
	require.NoError(t, err)
	_, err = part.Write([]byte("dummy excel content"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return buf.Bytes(), writer.FormDataContentType()
}
