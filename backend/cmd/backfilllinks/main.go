// Comando puntual: enlaza (user_id) las filas de exam_official_result de los
// alumnos que entregaron antes de que se importaran los resultados oficiales.
// Reutiliza el mismo matching que el envío (examservice.FindOfficialResult) en
// vez de reimplementarlo en SQL, que perdería la normalización de nombres.
//
// Uso:
//
//	go run ./cmd/backfilllinks            # dry-run: solo cuenta e imprime
//	go run ./cmd/backfilllinks -apply     # escribe
//	go run ./cmd/backfilllinks -exam 7    # limita a un examen
package main

import (
	"flag"
	"log"

	"github.com/inscripcion-moodle/go-backend/internal/config"
	"github.com/inscripcion-moodle/go-backend/internal/models"
	examservice "github.com/inscripcion-moodle/go-backend/internal/services/exam"
	"github.com/inscripcion-moodle/go-backend/internal/storage"
)

func main() {
	apply := flag.Bool("apply", false, "escribe los cambios (por defecto dry-run)")
	examID := flag.Uint("exam", 0, "limita a un examen (0 = todos)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	db, err := storage.NewMariaDB(cfg)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	q := db.Preload("User").Order("id")
	if *examID != 0 {
		q = q.Where("exam_id = ?", *examID)
	}
	var submissions []models.UserExamSubmission
	if err := q.Find(&submissions).Error; err != nil {
		log.Fatalf("load submissions: %v", err)
	}

	var linked, alreadyLinked, noMatch, taken int
	for _, sub := range submissions {
		var count int64
		if err := db.Model(&models.ExamOfficialResult{}).
			Where("exam_id = ? AND user_id = ?", sub.ExamID, sub.UserID).
			Count(&count).Error; err != nil {
			log.Fatalf("count linked: %v", err)
		}
		if count > 0 {
			alreadyLinked++
			continue
		}

		official, err := examservice.FindOfficialResult(db, examservice.OfficialResultMatchRequest{
			ExamID:  sub.ExamID,
			Name:    sub.User.Name,
			Surname: sub.User.Surname,
			DNI:     sub.User.DNI,
		})
		if err != nil {
			log.Fatalf("match submission %d: %v", sub.ID, err)
		}
		if official == nil {
			noMatch++
			continue
		}
		if official.UserID != nil {
			taken++
			continue
		}

		linked++
		if !*apply {
			continue
		}
		// El WHERE ... IS NULL evita pisar un link creado entre la lectura y la
		// escritura, igual que hace el flujo de envío.
		res := db.Model(&models.ExamOfficialResult{}).
			Where("id = ? AND user_id IS NULL", official.ID).
			Update("user_id", sub.UserID)
		if res.Error != nil {
			log.Fatalf("link official %d: %v", official.ID, res.Error)
		}
	}

	mode := "DRY-RUN"
	if *apply {
		mode = "APPLIED"
	}
	log.Printf("%s: %d entregas | %d ya linkadas | %d linkadas ahora | %d sin fila oficial | %d fila ya asignada a otro",
		mode, len(submissions), alreadyLinked, linked, noMatch, taken)
}
