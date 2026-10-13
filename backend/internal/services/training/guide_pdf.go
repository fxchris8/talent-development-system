package training

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/jung-kurt/gofpdf"
)

const (
	// Jumlah slide yang diperluas per panggilan LLM. Kecil supaya jawaban tidak terpotong.
	expansionBatchSize = 3
	// Panggilan LLM paralel maksimum saat memperluas slide.
	expansionConcurrency = 3
	expansionMaxTokens   = 12000
)

// jsonCompleter is the part of LLMClient the service needs; an interface so tests can fake it.
type jsonCompleter interface {
	CompleteJSON(ctx context.Context, system, prompt string, temperature float64, maxTokens int) (string, error)
}

// SlideExpansion is the study-guide text written for one PPT slide.
type SlideExpansion struct {
	Index        int      `json:"index"` // nomor slide konten, mulai dari 1 (urutan plan.Slides)
	Explanation  string   `json:"explanation"`
	Examples     []string `json:"examples"`
	KeyTakeaways []string `json:"key_takeaways"`
}

// pptSlideNumber is the slide number as shown in PowerPoint: the title slide is slide 1,
// so the first content slide (index 1) is slide 2.
func pptSlideNumber(contentIndex int) int { return contentIndex + 1 }

func (s *Service) buildExpansionPrompt(in GenerateInput, plan *Plan, indexes []int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, `Anda menulis PANDUAN BELAJAR (study guide) berbahasa Indonesia yang mendampingi sebuah presentasi PowerPoint training.
Panduan ini harus 100%% SESUAI dengan isi slide: jangan menambah topik di luar slide, jangan melewatkan satu pun poin slide.

KONTEKS TRAINING:
- Judul materi: %s
- Topik: %s
- Kompetensi: %s
- Level: %d
- Audiens: %s

SLIDE YANG HARUS DIPERLUAS:
`, plan.Title, in.TopikTraining, in.Kompetensi, in.Level, plan.Overview.Audience)

	for _, idx := range indexes {
		sl := plan.Slides[idx-1]
		fmt.Fprintf(&sb, "\n=== SLIDE %d (index %d) ===\nJudul: %s\n", pptSlideNumber(idx), idx, sl.Heading)
		for _, b := range sl.Bullets {
			if t := strings.TrimSpace(b); t != "" {
				fmt.Fprintf(&sb, "- %s\n", t)
			}
		}
		if n := strings.TrimSpace(sl.SpeakerNotes); n != "" {
			fmt.Fprintf(&sb, "Catatan fasilitator: %s\n", n)
		}
	}

	sb.WriteString(`
TUGAS: untuk SETIAP slide di atas tulis:
- "explanation": penjelasan lengkap 2-4 paragraf (150-250 kata), pisahkan paragraf dengan baris kosong. Jelaskan SEMUA poin slide secara berurutan.
- "examples": 1-2 contoh konkret yang relevan dengan dunia pelayaran/maritim.
- "key_takeaways": tepat 3 butir kesimpulan singkat.

OUTPUT: hanya JSON valid, tanpa markdown. "index" harus sama persis dengan index slide:
{"slides":[{"index":1,"explanation":"...","examples":["..."],"key_takeaways":["...","...","..."]}]}`)
	return sb.String()
}

func (s *Service) expandBatch(ctx context.Context, in GenerateInput, plan *Plan, indexes []int) (map[int]SlideExpansion, error) {
	system := "You are an expert educational content writer. Always respond with valid JSON only, no markdown formatting."
	raw, err := s.llm.CompleteJSON(ctx, system, s.buildExpansionPrompt(in, plan, indexes), 0.6, expansionMaxTokens)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Slides []SlideExpansion `json:"slides"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("gagal parsing penjelasan slide: %w", err)
	}

	wanted := make(map[int]bool, len(indexes))
	for _, idx := range indexes {
		wanted[idx] = true
	}
	out := make(map[int]SlideExpansion, len(indexes))
	for _, e := range parsed.Slides {
		if wanted[e.Index] && strings.TrimSpace(e.Explanation) != "" {
			out[e.Index] = e
		}
	}
	return out, nil
}

// expandSlides asks the LLM for study-guide text for every slide of the plan, a few slides at a time.
// A batch that fails is logged and skipped: those slides still appear in the PDF with their own
// bullets and speaker notes, so the guide never silently loses a slide.
func (s *Service) expandSlides(ctx context.Context, in GenerateInput, plan *Plan) map[int]SlideExpansion {
	result := make(map[int]SlideExpansion, len(plan.Slides))
	if s.llm == nil || len(plan.Slides) == 0 {
		return result
	}

	var batches [][]int
	for start := 1; start <= len(plan.Slides); start += expansionBatchSize {
		end := start + expansionBatchSize - 1
		if end > len(plan.Slides) {
			end = len(plan.Slides)
		}
		batch := make([]int, 0, expansionBatchSize)
		for i := start; i <= end; i++ {
			batch = append(batch, i)
		}
		batches = append(batches, batch)
	}

	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		sem = make(chan struct{}, expansionConcurrency)
	)
	for _, batch := range batches {
		wg.Add(1)
		sem <- struct{}{}
		go func(batch []int) {
			defer wg.Done()
			defer func() { <-sem }()

			got, err := s.expandBatch(ctx, in, plan, batch)
			if err != nil {
				s.log.WithError(err).Warnf("Gagal memperluas slide %v untuk PDF, memakai isi slide apa adanya", batch)
				return
			}
			mu.Lock()
			for idx, e := range got {
				result[idx] = e
			}
			mu.Unlock()
		}(batch)
	}
	wg.Wait()

	s.log.Infof("PDF guide: %d/%d slide berhasil diperluas", len(result), len(plan.Slides))
	return result
}

// buildGuidePDF writes the study guide: every slide of the PPT, in order and under the same slide
// number, followed by the activities, assessment and references of the plan.
func (s *Service) buildGuidePDF(plan *Plan, expansions map[int]SlideExpansion, filename string) error {
	return s.newGuidePDF(plan, expansions).OutputFileAndClose(filename)
}

const (
	pdfIndent   = 4.0
	pdfBulletW  = 5.0
	pdfLineH    = 6.0
	pdfBodySize = 11.0
)

var (
	pdfBlue = [3]int{0, 51, 102}
	pdfSub  = [3]int{0, 102, 204}
	pdfGrey = [3]int{90, 90, 90}
)

func cleanPDFText(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\r", ""))
}

func (s *Service) newGuidePDF(plan *Plan, expansions map[int]SlideExpansion) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddUTF8Font("DejaVu", "", "fonts/DejaVuSans.ttf")
	pdf.AddUTF8Font("DejaVu", "B", "fonts/DejaVuSans-Bold.ttf")

	pageW, _ := pdf.GetPageSize()
	left, _, right, _ := pdf.GetMargins()
	usable := pageW - left - right

	color := func(c [3]int) { pdf.SetTextColor(c[0], c[1], c[2]) }
	black := func() { pdf.SetTextColor(0, 0, 0) }

	heading := func(text string, size float64) {
		pdf.SetFont("DejaVu", "B", size)
		color(pdfBlue)
		pdf.MultiCell(0, size*0.55, text, "", "L", false)
		pdf.Ln(2)
		black()
	}
	label := func(text string) {
		pdf.Ln(2)
		pdf.SetFont("DejaVu", "B", 12)
		color(pdfSub)
		pdf.MultiCell(0, 7, text, "", "L", false)
		black()
	}
	paragraph := func(text string, size float64) {
		pdf.SetFont("DejaVu", "", size)
		for _, p := range strings.Split(cleanPDFText(text), "\n") {
			if p = strings.TrimSpace(p); p == "" {
				pdf.Ln(2)
				continue
			}
			pdf.SetX(left + pdfIndent)
			pdf.MultiCell(usable-pdfIndent, pdfLineH, p, "", "L", false)
			pdf.Ln(1)
		}
	}
	bullet := func(text string) {
		text = cleanPDFText(text)
		if text == "" {
			return
		}
		pdf.SetFont("DejaVu", "", pdfBodySize)
		x := left + pdfIndent
		pdf.SetX(x)
		pdf.CellFormat(pdfBulletW, pdfLineH, "•", "", 0, "L", false, 0, "")
		pdf.SetX(x + pdfBulletW)
		pdf.MultiCell(usable-pdfIndent-pdfBulletW, pdfLineH, text, "", "L", false)
	}

	// ---- cover
	pdf.AddPage()
	pdf.SetFont("DejaVu", "B", 22)
	color(pdfBlue)
	pdf.MultiCell(0, 12, plan.Title, "", "C", false)
	pdf.Ln(4)
	pdf.SetFont("DejaVu", "", 13)
	pdf.SetTextColor(80, 80, 80)
	pdf.MultiCell(0, 8, "Panduan Pembelajaran - pendamping presentasi", "", "C", false)
	pdf.Ln(8)
	black()
	pdf.SetFont("DejaVu", "", 12)
	if plan.Overview.Duration != "" {
		pdf.MultiCell(0, 7, "Durasi: "+plan.Overview.Duration, "", "L", false)
	}
	if plan.Overview.Audience != "" {
		pdf.MultiCell(0, 7, "Audiens: "+plan.Overview.Audience, "", "L", false)
	}
	if len(plan.Overview.Goals) > 0 {
		label("Tujuan Pembelajaran")
		for _, g := range plan.Overview.Goals {
			bullet(g)
		}
	}
	if len(plan.Overview.Outcomes) > 0 {
		label("Hasil yang Diharapkan")
		for _, o := range plan.Overview.Outcomes {
			bullet(o)
		}
	}

	// ---- table of contents (same numbering as the PPT)
	pdf.AddPage()
	heading("Daftar Isi", 18)
	pdf.SetFont("DejaVu", "", 12)
	for i, sl := range plan.Slides {
		pdf.SetX(left + pdfIndent)
		pdf.MultiCell(usable-pdfIndent, 7, fmt.Sprintf("Slide %d - %s", pptSlideNumber(i+1), sl.Heading), "", "L", false)
	}
	extra := []string{}
	if len(plan.Activities) > 0 {
		extra = append(extra, "Aktivitas & Latihan")
	}
	if len(plan.Assessment) > 0 {
		extra = append(extra, "Assessment & Evaluasi")
	}
	if len(plan.References) > 0 {
		extra = append(extra, "Referensi")
	}
	for _, e := range extra {
		pdf.SetX(left + pdfIndent)
		pdf.MultiCell(usable-pdfIndent, 7, e, "", "L", false)
	}

	// ---- one section per PPT slide
	for i, sl := range plan.Slides {
		idx := i + 1
		pdf.AddPage()
		heading(fmt.Sprintf("Slide %d - %s", pptSlideNumber(idx), sl.Heading), 16)

		label("Isi Slide")
		for _, b := range sl.Bullets {
			t := cleanPDFText(b)
			switch {
			case t == "":
				pdf.Ln(2)
			case strings.HasPrefix(t, "•") || strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* "):
				// pola prompt: "• Judul sub-bagian" diikuti paragraf penjelasan
				pdf.Ln(1)
				pdf.SetFont("DejaVu", "B", 11.5)
				pdf.SetX(left + pdfIndent)
				pdf.MultiCell(usable-pdfIndent, pdfLineH, strings.TrimSpace(strings.TrimLeft(t, "•-* ")), "", "L", false)
			default:
				paragraph(t, pdfBodySize)
			}
		}

		if e, ok := expansions[idx]; ok {
			label("Penjelasan Lengkap")
			paragraph(e.Explanation, pdfBodySize)
			if len(e.Examples) > 0 {
				label("Contoh")
				for _, ex := range e.Examples {
					bullet(ex)
				}
			}
			if len(e.KeyTakeaways) > 0 {
				label("Kesimpulan Utama")
				for _, k := range e.KeyTakeaways {
					bullet(k)
				}
			}
		}

		if notes := cleanPDFText(sl.SpeakerNotes); notes != "" {
			label("Catatan Fasilitator")
			pdf.SetTextColor(pdfGrey[0], pdfGrey[1], pdfGrey[2])
			paragraph(notes, 10)
			black()
		}
	}

	// ---- sections that also exist in the PPT
	if len(plan.Activities) > 0 {
		pdf.AddPage()
		heading("Aktivitas & Latihan", 16)
		for n, a := range plan.Activities {
			pdf.SetFont("DejaVu", "B", 12)
			pdf.MultiCell(0, 7, fmt.Sprintf("%d. %s (%s)", n+1, a.Title, a.Time), "", "L", false)
			paragraph(a.Instructions, pdfBodySize)
			pdf.Ln(2)
		}
	}
	if len(plan.Assessment) > 0 {
		pdf.AddPage()
		heading("Assessment & Evaluasi", 16)
		for n, qa := range plan.Assessment {
			pdf.SetFont("DejaVu", "B", 12)
			pdf.MultiCell(0, 7, fmt.Sprintf("Q%d: %s", n+1, qa.Question), "", "L", false)
			paragraph("A: "+qa.Answer, pdfBodySize)
			pdf.Ln(2)
		}
	}
	if len(plan.References) > 0 {
		pdf.AddPage()
		heading("Referensi", 16)
		for _, r := range plan.References {
			bullet(r)
		}
	}

	return pdf
}

// expandedSlideNumbers lists, for logging/tests, which PPT slide numbers got an expansion.
func expandedSlideNumbers(expansions map[int]SlideExpansion) []int {
	nums := make([]int, 0, len(expansions))
	for idx := range expansions {
		nums = append(nums, pptSlideNumber(idx))
	}
	sort.Ints(nums)
	return nums
}
