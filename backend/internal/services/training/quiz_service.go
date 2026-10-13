package training

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/sirupsen/logrus"
)

type QuizService struct {
	log     *logrus.Logger
	llm     *LLMClient
	pubBase string
	outDir  string
}

func NewQuizService(log *logrus.Logger, llm *LLMClient, pubBase string) *QuizService {
	if pubBase == "" {
		pubBase = "http://localhost:8080"
	}

	outDir := "./public/materi"
	_ = os.MkdirAll(outDir, 0o755)

	return &QuizService{
		log:     log,
		llm:     llm,
		pubBase: strings.TrimRight(pubBase, "/"),
		outDir:  outDir,
	}
}

type QuizQuestion struct {
	Number   int      `json:"number"`
	Question string   `json:"question"`
	OptionA  string   `json:"option_a"`
	OptionB  string   `json:"option_b"`
	OptionC  string   `json:"option_c"`
	OptionD  string   `json:"option_d"`
	Answer   string   `json:"answer"`
}

type QuizResponse struct {
	Questions []QuizQuestion `json:"questions"`
}

func (s *QuizService) GenerateQuizFromMaterial(ctx context.Context, materialContent, topicTitle, competency string) (string, error) {
	s.log.WithFields(logrus.Fields{
		"topic":      topicTitle,
		"competency": competency,
	}).Info("Starting quiz generation from material")

	questions, err := s.callLLMForQuiz(ctx, materialContent, topicTitle, competency)
	if err != nil {
		return "", fmt.Errorf("failed to generate quiz questions: %w", err)
	}

	if len(questions) == 0 {
		return "", errors.New("no questions generated")
	}

	pdfPath, err := s.buildQuizPDF(questions, topicTitle, competency)
	if err != nil {
		return "", fmt.Errorf("failed to build quiz PDF: %w", err)
	}

	publicURL := s.pubBase + "/files/materi/" + filepath.Base(pdfPath)
	s.log.WithField("url", publicURL).Info("Quiz PDF generated successfully")

	return publicURL, nil
}

func (s *QuizService) callLLMForQuiz(ctx context.Context, materialContent, topicTitle, competency string) ([]QuizQuestion, error) {
	prompt := s.buildQuizPrompt(materialContent, topicTitle, competency)

	content, err := s.llm.CompleteJSON(ctx, "You are an expert assessment writer. Respond ONLY with valid JSON, no markdown formatting.", prompt, 0.7, 16000)
	if err != nil {
		return nil, err
	}

	var quizResp QuizResponse
	if err := json.Unmarshal([]byte(content), &quizResp); err != nil {
		s.log.WithError(err).WithField("content", content).Error("Failed to parse quiz JSON")
		return nil, fmt.Errorf("failed to parse quiz response: %w", err)
	}

	return quizResp.Questions, nil
}

func (s *QuizService) buildQuizPrompt(materialContent, topicTitle, competency string) string {
	return fmt.Sprintf(`Anda adalah ahli dalam membuat soal evaluasi training. Berdasarkan materi training berikut, buatlah 20 soal pilihan ganda berkualitas tinggi.

MATERI TRAINING:
Topik: %s
Kompetensi: %s

Konten Materi:
%s

INSTRUKSI:
1. Buat TEPAT 20 soal pilihan ganda (A, B, C, D)
2. Soal harus menguji pemahaman konsep dari materi, bukan hanya hafalan
3. Setiap soal harus jelas dan tidak ambigu
4. Pilihan jawaban harus masuk akal dan menantang
5. Sertakan jawaban yang benar untuk setiap soal (A/B/C/D)
6. Soal harus progresif: mulai dari basic understanding, ke pemahaman mendalam, hingga aplikasi konsep
7. Variasikan tingkat kesulitan soal: 30%% mudah, 50%% sedang, 20%% sulit
8. Pastikan soal mencakup semua aspek penting dari materi

Format output HARUS dalam JSON valid berikut (tanpa markdown, tanpa backticks):
{
  "questions": [
    {
      "number": 1,
      "question": "Teks pertanyaan yang jelas dan lengkap?",
      "option_a": "Pilihan A",
      "option_b": "Pilihan B", 
      "option_c": "Pilihan C",
      "option_d": "Pilihan D",
      "answer": "A"
    }
  ]
}

PENTING: Output HARUS JSON murni, JANGAN tambahkan markdown atau backticks!`, topicTitle, competency, materialContent)
}

func (s *QuizService) buildQuizPDF(questions []QuizQuestion, topicTitle, competency string) (string, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)

	pdf.AddUTF8Font("DejaVu", "", "fonts/DejaVuSans.ttf")
	pdf.AddUTF8Font("DejaVu", "B", "fonts/DejaVuSans-Bold.ttf")

	pdf.AddPage()

	pdf.SetFont("DejaVu", "B", 18)
	pdf.SetTextColor(31, 41, 55)
	pdf.CellFormat(0, 12, "PRETEST & POSTTEST", "", 1, "C", false, 0, "")
	
	pdf.Ln(3)
	pdf.SetFont("DejaVu", "B", 14)
	pdf.SetTextColor(59, 130, 246)
	pdf.MultiCell(0, 8, topicTitle, "", "C", false)
	
	pdf.Ln(2)
	pdf.SetFont("DejaVu", "", 10)
	pdf.SetTextColor(107, 114, 128)
	pdf.CellFormat(0, 6, "Kompetensi: "+competency, "", 1, "C", false, 0, "")
	
	pdf.Ln(5)
	pdf.SetDrawColor(229, 231, 235)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(8)

	for _, q := range questions {
		if pdf.GetY() > 250 {
			pdf.AddPage()
		}

		pdf.SetFont("DejaVu", "B", 11)
		pdf.SetTextColor(31, 41, 55)
		questionText := fmt.Sprintf("%d. %s", q.Number, q.Question)
		pdf.MultiCell(0, 6, questionText, "", "L", false)
		pdf.Ln(3)

		options := []struct {
			label string
			text  string
		}{
			{"A", q.OptionA},
			{"B", q.OptionB},
			{"C", q.OptionC},
			{"D", q.OptionD},
		}

		pdf.SetFont("DejaVu", "", 10)
		pdf.SetTextColor(55, 65, 81)
		
		for _, opt := range options {
			optionText := fmt.Sprintf("   %s. %s", opt.label, opt.text)
			pdf.MultiCell(0, 5, optionText, "", "L", false)
			pdf.Ln(1)
		}

		pdf.Ln(5)
	}

	pdf.AddPage()
	pdf.SetFont("DejaVu", "B", 14)
	pdf.SetTextColor(31, 41, 55)
	pdf.CellFormat(0, 10, "KUNCI JAWABAN", "", 1, "C", false, 0, "")
	pdf.Ln(5)

	pdf.SetFont("DejaVu", "", 10)
	pdf.SetTextColor(55, 65, 81)
	
	for i, q := range questions {
		answerText := fmt.Sprintf("%d. %s", q.Number, q.Answer)
		pdf.CellFormat(0, 6, answerText, "", 1, "L", false, 0, "")
		
		if (i+1)%5 == 0 {
			pdf.Ln(2)
		}
	}

	timestamp := time.Now().Format("20060102-150405")
	safeTitle := strings.ReplaceAll(topicTitle, " ", "_")
	safeTitle = strings.ReplaceAll(safeTitle, "/", "-")
	
	filename := fmt.Sprintf("quiz_%s_%s.pdf", safeTitle, timestamp)
	outPath := filepath.Join(s.outDir, filename)

	if err := pdf.OutputFileAndClose(outPath); err != nil {
		return "", err
	}

	return outPath, nil
}

func (s *QuizService) DeleteOldQuizFile(oldURL string) error {
	if oldURL == "" {
		return nil
	}

	prefix := s.pubBase + "/files/materi/"
	if !strings.HasPrefix(oldURL, prefix) {
		return nil
	}

	filename := strings.TrimPrefix(oldURL, prefix)
	fullPath := filepath.Join(s.outDir, filename)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil
	}

	return os.Remove(fullPath)
}
