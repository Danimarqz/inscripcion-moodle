package admin

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/inscripcion-moodle/go-backend/internal/scoring"
	examservice "github.com/inscripcion-moodle/go-backend/internal/services/exam"
	"github.com/xuri/excelize/v2"
	"time"
)

// aptoLabel renders a pass/fail flag as the Spanish "Sí"/"No" used in the export.
func aptoLabel(passed bool) string {
	if passed {
		return "Sí"
	}
	return "No"
}

func (s *Service) ExportSubmissionsAnalysis(examID uint, search, orderBy, orderDir string, moodleSynced *bool, resultType *string) (*bytes.Buffer, error) {
	queryCtx, queryCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer queryCancel()
	exam, err := s.examRepo.FindExamByID(queryCtx, s.db, examID)
	if err != nil {
		return nil, err
	}

	const maxLimit = 50000
	orderClause := buildSubmissionOrder(orderBy, orderDir)
	queryCtx, queryCancel = context.WithTimeout(context.Background(), 3*time.Minute)
	defer queryCancel()
	submissions, err := s.submissionRepo.List(queryCtx, s.db, examID, maxLimit, 0, search, orderClause, moodleSynced, resultType, true)
	if err == nil && len(submissions) >= maxLimit {
		log.Printf("WARNING: ExportSubmissionsAnalysis exam_id=%d reached maxLimit=%d (got %d submissions). Data may be truncated.", examID, maxLimit, len(submissions))
	}
	if err != nil {
		return nil, err
	}

	questionsMap := make(map[uint]models.Question)
	sortedQuestions := make([]models.Question, len(exam.Questions))
	copy(sortedQuestions, exam.Questions)
	for i := range sortedQuestions {
		for j := i + 1; j < len(sortedQuestions); j++ {
			if sortedQuestions[i].Name > sortedQuestions[j].Name {
				sortedQuestions[i], sortedQuestions[j] = sortedQuestions[j], sortedQuestions[i]
			}
		}
	}

	for _, q := range exam.Questions {
		questionsMap[q.ID] = q
	}

	// Group dimension: map each question to its group name (empty when the exam
	// has no groups). Answers come from AnswersData (jsonb) only, same as scoring.
	hasGroups := len(exam.Groups) > 0
	groupNameByQuestion := make(map[uint]string)
	for _, g := range exam.Groups {
		for _, q := range exam.Questions {
			if q.GroupID != nil && *q.GroupID == g.ID {
				groupNameByQuestion[q.ID] = g.Name
			}
		}
	}

	failures := make(map[uint]int)
	totalAttempts := len(submissions)

	for _, sub := range submissions {
		userAnswers := make(map[uint]string)
		if sub.AnswersData != nil {
			userAnswers = map[uint]string(*sub.AnswersData)
		}

		for _, q := range exam.Questions {
			if !q.IsActive || q.IsCancelled {
				continue
			}
			userAns, ok := userAnswers[q.ID]

			normalizedUserAns := strings.ToUpper(strings.TrimSpace(userAns))
			normalizedCorrect := strings.ToUpper(strings.TrimSpace(q.CorrectOption))

			if !ok || normalizedUserAns != normalizedCorrect {
				failures[q.ID]++
			}
		}
	}

	f := excelize.NewFile()
	sheetStats := "Estadísticas"
	f.SetSheetName("Sheet1", sheetStats)

	f.SetCellValue(sheetStats, "A1", "Pregunta")
	f.SetCellValue(sheetStats, "B1", "Fallos")
	f.SetCellValue(sheetStats, "C1", "Porcentaje Fallos")
	if hasGroups {
		f.SetCellValue(sheetStats, "D1", "Grupo")
	}

	for i, q := range sortedQuestions {
		if !q.IsActive || q.IsCancelled {
			continue
		}
		row := i + 2
		failCount := failures[q.ID]
		percent := 0.0
		if totalAttempts > 0 {
			percent = float64(failCount) / float64(totalAttempts)
		}

		f.SetCellValue(sheetStats, fmt.Sprintf("A%d", row), fmt.Sprintf("Q%d", q.Name))
		f.SetCellValue(sheetStats, fmt.Sprintf("B%d", row), failCount)
		f.SetCellValue(sheetStats, fmt.Sprintf("C%d", row), percent)
	}

	currentRow := 2
	for _, q := range sortedQuestions {
		if !q.IsActive || q.IsCancelled {
			continue
		}
		failCount := failures[q.ID]
		percent := 0.0
		if totalAttempts > 0 {
			percent = float64(failCount) / float64(totalAttempts)
		}
		f.SetCellValue(sheetStats, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("%d", q.Name))
		f.SetCellValue(sheetStats, fmt.Sprintf("B%d", currentRow), failCount)
		f.SetCellValue(sheetStats, fmt.Sprintf("C%d", currentRow), percent)
		if hasGroups {
			f.SetCellValue(sheetStats, fmt.Sprintf("D%d", currentRow), groupNameByQuestion[q.ID])
		}
		currentRow++
	}
	lastDataRow := currentRow - 1

	if lastDataRow >= 2 {
		if err := f.AddChart(sheetStats, "E2", &excelize.Chart{
			Type: excelize.Col,
			Series: []excelize.ChartSeries{
				{
					Name:       "Estadísticas!$B$1",
					Categories: fmt.Sprintf("Estadísticas!$A$2:$A$%d", lastDataRow),
					Values:     fmt.Sprintf("Estadísticas!$B$2:$B$%d", lastDataRow),
				},
			},
			Title: excelize.ChartTitle{
				Paragraph: []excelize.RichTextRun{
					{
						Text: "Fallos por Pregunta",
					},
				},
			},
		}); err != nil {
			return nil, err
		}
	}

	sheetData := "Datos"
	f.NewSheet(sheetData)

	headers := []string{"ID", "Nombre", "Apellidos", "Email", "DNI", "Nota", "Percentil", "Fecha", "Tipo"}
	for _, q := range sortedQuestions {
		headers = append(headers, fmt.Sprintf("P%d - %s", q.Name, q.CorrectOption))
	}
	// Per-group columns (only for grouped exams), appended after the answer
	// columns: one "Nota"/"Apto" pair per group plus a global "Apto".
	if hasGroups {
		headers = append(headers, "Apto")
		for _, g := range exam.Groups {
			headers = append(headers, fmt.Sprintf("%s Nota", g.Name), fmt.Sprintf("%s Apto", g.Name))
		}
	}
	groupColStart := 10 + len(sortedQuestions)

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetData, cell, h)
	}

	for i, sub := range submissions {
		row := i + 2

		f.SetCellValue(sheetData, fmt.Sprintf("A%d", row), sub.ID)
		f.SetCellValue(sheetData, fmt.Sprintf("B%d", row), sub.User.Name)
		f.SetCellValue(sheetData, fmt.Sprintf("C%d", row), sub.User.Surname)
		f.SetCellValue(sheetData, fmt.Sprintf("D%d", row), sub.User.Email)
		f.SetCellValue(sheetData, fmt.Sprintf("E%d", row), sub.User.DNI)

		score := 0.0
		if sub.Score != nil {
			score = *sub.Score
		}

		f.SetCellValue(sheetData, fmt.Sprintf("F%d", row), score)

		percentile := 0.0
		if sub.Percentile != nil {
			percentile = *sub.Percentile
		}
		f.SetCellValue(sheetData, fmt.Sprintf("G%d", row), percentile)

		f.SetCellValue(sheetData, fmt.Sprintf("H%d", row), sub.SubmittedAt.Format("2006-01-02 15:04:05"))
		f.SetCellValue(sheetData, fmt.Sprintf("I%d", row), sub.SelectedResultType)

		userAnswers := make(map[uint]string)
		if sub.AnswersData != nil {
			userAnswers = map[uint]string(*sub.AnswersData)
		}

		colIndex := 10
		for _, q := range sortedQuestions {
			ans, ok := userAnswers[q.ID]
			cell, _ := excelize.CoordinatesToCellName(colIndex, row)
			if ok {
				f.SetCellValue(sheetData, cell, ans)
			} else {
				f.SetCellValue(sheetData, cell, "")
			}
			colIndex++
		}

		if hasGroups {
			// Per-group Nota/Apto + global Apto, computed with the same scoring +
			// mode selection as GetSubmissionBreakdown (grouped absolute vs Xunta).
			var bd *examservice.ScoreBreakdown
			if exam.ScoringMode == "xunta" {
				bd, err = examservice.CalculateGroupedXuntaBreakdown(exam.Groups, exam.Questions, userAnswers)
			} else {
				bd, err = examservice.CalculateGroupedBreakdown(exam.Groups, exam.Questions, userAnswers, examservice.ScoringConfigFromExam(exam).WrongBlockSize)
			}
			if err != nil {
				return nil, err
			}
			outcomes := map[uint]scoring.GroupOutcome{}
			if bd != nil {
				for _, g := range bd.Groups {
					outcomes[g.GroupID] = g
				}
			}
			aptoCell, _ := excelize.CoordinatesToCellName(groupColStart, row)
			f.SetCellValue(sheetData, aptoCell, aptoLabel(bd != nil && scoring.GroupedResult{Groups: bd.Groups}.AllEliminatoryPassed()))
			col := groupColStart + 1
			for _, g := range exam.Groups {
				o := outcomes[g.ID]
				notaCell, _ := excelize.CoordinatesToCellName(col, row)
				f.SetCellValue(sheetData, notaCell, o.Score)
				aptoGCell, _ := excelize.CoordinatesToCellName(col+1, row)
				f.SetCellValue(sheetData, aptoGCell, aptoLabel(o.Passed))
				col += 2
			}
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf, nil
}
