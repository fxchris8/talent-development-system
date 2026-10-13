package trainingController

import (
	"context"
	"net/http"
	"time"

	"backend/internal/services"
	"backend/internal/services/training"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type TrainingController struct {
	dbSvc   *services.TrainingService
	aiSvc   *training.Service
	quizSvc *training.QuizService
	audioSvc *training.NarrationAudioService
	log     *logrus.Logger
}

func NewTrainingController(
	dbSvc *services.TrainingService,
	aiSvc *training.Service,
	quizSvc *training.QuizService,
	audioSvc *training.NarrationAudioService,
	log *logrus.Logger,
) *TrainingController {
	return &TrainingController{
		dbSvc:   dbSvc,
		aiSvc:   aiSvc,
		quizSvc: quizSvc,
		audioSvc: audioSvc,
		log:     log,
	}
}

// ======================
// CRUD Endpoint
// ======================
func (h *TrainingController) FindAll(c *gin.Context) {
	res, err := h.dbSvc.FindAll()
	if err != nil {
		h.log.WithError(err).Error("gagal mengambil data training")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":   http.StatusInternalServerError,
			"status": "ERROR",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, res)
}

// ======================
// AI Generator Endpoint
// ======================
func (h *TrainingController) Generate(c *gin.Context) {
	var req struct {
		No               int    `json:"no" binding:"required"`
		Kode             string `json:"kode" binding:"required"`
		TopikTraining    string `json:"topik_training" binding:"required"`
		Kompetensi       string `json:"kompetensi" binding:"required"`
		Referensi        string `json:"referensi"`
		Level            int    `json:"lvl"`
		Tools            string `json:"tools_training"`
		DeskripsiPerilaku string `json:"deskripsi_perilaku"`
		OldFileURL       string `json:"old_file_url"` // untuk regenerate PPTX
		OldPdfURL        string `json:"old_pdf_url"`  // untuk regenerate PDF
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Request tidak valid: " + err.Error(),
		})
		return
	}

	// Generate materi (PPTX + PDF) dengan pass oldFileURL untuk delete file lama
	pptxLink, pdfLink, meta, err := h.aiSvc.GenerateAndBuildPPTX(c.Request.Context(), training.GenerateInput{
		Kode:             req.Kode,
		TopikTraining:    req.TopikTraining,
		Kompetensi:       req.Kompetensi,
		Referensi:        req.Referensi,
		Level:            req.Level,
		Tools:            req.Tools,
		DeskripsiPerilaku: req.DeskripsiPerilaku,
	}, req.OldFileURL, req.OldPdfURL)

	if err != nil {
		h.log.WithError(err).Error("Gagal generate materi training")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal membuat materi training: " + err.Error(),
		})
		return
	}

	// Save both links to database
	if err := h.dbSvc.UpdateGeneratedFileURL(req.No, pptxLink, pdfLink); err != nil {
		h.log.WithError(err).Warn("Gagal menyimpan link file ke database, tapi file sudah digenerate")
		// Not returning error, karena file sudah berhasil digenerate
	}

	// The narration belongs to one PPTX: drop the old one and prepare the new one in the
	// background so "Dengarkan" can play right away.
	if req.OldFileURL != "" {
		h.audioSvc.Remove(req.OldFileURL)
	}
	go func(pptx string) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
		defer cancel()
		if _, err := h.audioSvc.Ensure(ctx, pptx); err != nil {
			h.log.WithError(err).Warn("Gagal menyiapkan audio narasi di latar belakang")
		}
	}(pptxLink)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Materi training berhasil digenerate (PPTX + PDF).",
		"pptx_link": pptxLink,
		"pdf_link":  pdfLink,
		"meta":     meta,
	})
}

func (h *TrainingController) GenerateQuiz(c *gin.Context) {
	var req struct {
		No               int    `json:"no" binding:"required"`
		Kode             string `json:"kode" binding:"required"`
		TopikTraining    string `json:"topik_training" binding:"required"`
		Kompetensi       string `json:"kompetensi" binding:"required"`
		MaterialContent  string `json:"material_content" binding:"required"`
		OldQuizURL       string `json:"old_quiz_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Request tidak valid: " + err.Error(),
		})
		return
	}

	if req.OldQuizURL != "" {
		if err := h.quizSvc.DeleteOldQuizFile(req.OldQuizURL); err != nil {
			h.log.WithError(err).Warn("Gagal menghapus quiz file lama")
		}
	}

	quizLink, err := h.quizSvc.GenerateQuizFromMaterial(c.Request.Context(), req.MaterialContent, req.TopikTraining, req.Kompetensi)
	if err != nil {
		h.log.WithError(err).Error("Gagal generate quiz")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal membuat quiz: " + err.Error(),
		})
		return
	}

	if err := h.dbSvc.UpdateGeneratedQuizURL(req.No, quizLink); err != nil {
		h.log.WithError(err).Warn("Gagal menyimpan link quiz ke database, tapi file sudah digenerate")
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Quiz berhasil digenerate.",
		"quiz_link": quizLink,
	})
}

// Narration returns the text of every slide of a generated PPT so the browser can read it aloud.
func (h *TrainingController) Narration(c *gin.Context) {
	var req struct {
		PptxURL string `json:"pptx_url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Request tidak valid: " + err.Error(),
		})
		return
	}

	slides, err := h.aiSvc.SlideNarration(req.PptxURL)
	if err != nil {
		h.log.WithError(err).Error("Gagal mengambil narasi PPT")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil narasi: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"slides": slides,
	})
}

// NarrationAudio returns the URL of the spoken (Indonesian) narration of a generated PPT,
// creating the audio on first use.
func (h *TrainingController) NarrationAudio(c *gin.Context) {
	var req struct {
		PptxURL string `json:"pptx_url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File PPT belum tersedia untuk materi ini.",
		})
		return
	}

	// Not tied to the request context: closing the tab must not waste a half-made audio.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	audioURL, err := h.audioSvc.Ensure(ctx, req.PptxURL)
	if err != nil {
		h.log.WithError(err).Error("Gagal membuat audio narasi")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal membuat suara: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"audio_url": audioURL,
	})
}
