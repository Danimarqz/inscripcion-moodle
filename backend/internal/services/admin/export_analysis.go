package admin

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/xuri/excelize/v2"
)

func (s *Service) ExportSubmissionsAnalysis(examID uint, search, orderBy, orderDir string, moodleSynced *bool, resultType *string) (*bytes.Buffer, error) {
	exam, err := s.examRepo.FindExamByID(context.Background(), s.db, examID)
	if err != nil {
		return nil, err
	}

	const maxLimit = 1000000
	orderClause := buildSubmissionOrder(orderBy, orderDir)
	submissions, err := s.submissionRepo.List(context.Background(), s.db, examID, true, maxLimit, 0, search, orderClause, moodleSynced, resultType)
	if err != nil {
		return nil, err
	}

	questionsMap := make(map[uint]models.Question)
	sortedQuestions := make([]models.Question, len(exam.Questions))
	copy(sortedQuestions, exam.Questions)
	for i := 0; i < len(sortedQuestions); i++ {
		for j := i + 1; j < len(sortedQuestions); j++ {
			if sortedQuestions[i].Name > sortedQuestions[j].Name {
				sortedQuestions[i], sortedQuestions[j] = sortedQuestions[j], sortedQuestions[i]
			}
		}
	}

	for _, q := range exam.Questions {
		questionsMap[q.ID] = q
	}

	failures := make(map[uint]int)
	totalAttempts := len(submissions)

	for _, sub := range submissions {
		userAnswers := make(map[uint]string)
		for _, ans := range sub.Answers {
			userAnswers[ans.QuestionID] = ans.Answer
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
			Title: []excelize.RichTextRun{
				{
					Text: "Fallos por Pregunta",
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
		headers = append(headers, fmt.Sprintf("P%d (Correcta: %s)", q.Name, q.CorrectOption))
	}

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
		for _, ans := range sub.Answers {
			userAnswers[ans.QuestionID] = ans.Answer
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
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf, nil
}
