package training

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
)

var indexRe = regexp.MustCompile(`\(index (\d+)\)`)

// fakeLLM answers the expansion prompt for exactly the slide indexes mentioned in it.
type fakeLLM struct {
	mu       sync.Mutex
	calls    [][]int
	failFor  int // batch containing this index returns an error
	truncate int // batch containing this index returns cut-off JSON
}

func (f *fakeLLM) CompleteJSON(_ context.Context, _, prompt string, _ float64, _ int) (string, error) {
	var idx []int
	for _, m := range indexRe.FindAllStringSubmatch(prompt, -1) {
		n, _ := strconv.Atoi(m[1])
		idx = append(idx, n)
	}
	f.mu.Lock()
	f.calls = append(f.calls, idx)
	f.mu.Unlock()

	for _, n := range idx {
		if n == f.failFor {
			return "", errors.New("llm down")
		}
		if n == f.truncate {
			return `{"slides":[{"index":` + strconv.Itoa(n) + `,"explanation":"terpot`, nil
		}
	}

	type out struct {
		Index        int      `json:"index"`
		Explanation  string   `json:"explanation"`
		Examples     []string `json:"examples"`
		KeyTakeaways []string `json:"key_takeaways"`
	}
	var slides []out
	for _, n := range idx {
		slides = append(slides, out{
			Index:        n,
			Explanation:  fmt.Sprintf("Penjelasan panjang untuk slide konten %d.\n\nParagraf kedua slide %d.", n, n),
			Examples:     []string{fmt.Sprintf("Contoh kapal %d", n)},
			KeyTakeaways: []string{"satu", "dua", "tiga"},
		})
	}
	b, _ := json.Marshal(map[string]any{"slides": slides})
	return string(b), nil
}

func testPlan(slideCount int) *Plan {
	plan := &Plan{
		Title:      "Dasar Kepemimpinan",
		Overview:   Overview{Goals: []string{"Memahami X"}, Outcomes: []string{"Mampu Y"}, Duration: "2 jam", Audience: "Perwira"},
		Activities: []Activity{{Title: "Diskusi", Time: "10 menit", Instructions: "Diskusikan kasus"}},
		Assessment: []QA{{Question: "Apa itu X?", Answer: "X adalah Y"}},
		References: []string{"Buku A"},
	}
	for i := 1; i <= slideCount; i++ {
		plan.Slides = append(plan.Slides, Slide{
			Heading:      "Heading " + strconv.Itoa(i),
			Bullets:      []string{"• Sub judul " + strconv.Itoa(i), "Isi paragraf slide " + strconv.Itoa(i), "", "Poin lain"},
			SpeakerNotes: "Catatan " + strconv.Itoa(i),
			SlideType:    "content",
		})
	}
	return plan
}

func TestExpandSlidesCoversEverySlideInBatches(t *testing.T) {
	llm := &fakeLLM{}
	svc := &Service{log: logrus.New(), llm: llm}
	plan := testPlan(12)

	got := svc.expandSlides(context.Background(), GenerateInput{TopikTraining: "Leadership"}, plan)

	if len(got) != 12 {
		t.Fatalf("expanded %d slides, want 12", len(got))
	}
	for i := 1; i <= 12; i++ {
		if !strings.Contains(got[i].Explanation, fmt.Sprintf("slide konten %d.", i)) {
			t.Errorf("slide %d got wrong/missing explanation: %q", i, got[i].Explanation)
		}
	}
	if len(llm.calls) != 4 {
		t.Errorf("want 4 batched LLM calls for 12 slides, got %d: %v", len(llm.calls), llm.calls)
	}
}

func TestExpandSlidesSurvivesFailedAndTruncatedBatches(t *testing.T) {
	t.Chdir("../../..") // fonts/ lives in the backend root
	// slide 2 -> batch {1,2,3} errors out; slide 8 -> batch {7,8,9} returns cut-off JSON
	svc := &Service{log: logrus.New(), llm: &fakeLLM{failFor: 2, truncate: 8}}
	plan := testPlan(12)

	got := svc.expandSlides(context.Background(), GenerateInput{}, plan)

	if len(got) != 6 {
		t.Fatalf("want 6 slides expanded (2 of 4 batches lost), got %d: %v", len(got), expandedSlideNumbers(got))
	}
	for _, lost := range []int{1, 2, 3, 7, 8, 9} {
		if _, ok := got[lost]; ok {
			t.Errorf("slide %d should not have an expansion", lost)
		}
	}

	// the PDF must still contain every slide even though some expansions are missing
	doc := svc.newGuidePDF(plan, got)
	if doc.Error() != nil {
		t.Fatalf("pdf error: %v", doc.Error())
	}
	// cover + toc + 12 slides + activities + assessment + references
	if min := 2 + 12 + 3; doc.PageCount() < min {
		t.Errorf("pdf has %d pages, want at least %d (one per slide)", doc.PageCount(), min)
	}
}

func TestGuidePDFWritesFileWithoutLLM(t *testing.T) {
	t.Chdir("../../..") // fonts/ lives in the backend root
	svc := &Service{log: logrus.New()}
	plan := testPlan(5)

	out := filepath.Join(t.TempDir(), "guide.pdf")
	if err := svc.buildPDF(plan, out); err != nil {
		t.Fatalf("buildPDF: %v", err)
	}
	info, err := os.Stat(out)
	if err != nil || info.Size() < 2000 {
		t.Fatalf("pdf not written properly: %v size=%v", err, info)
	}

	// no LLM configured: expandSlides must be a harmless no-op
	if got := svc.expandSlides(context.Background(), GenerateInput{}, plan); len(got) != 0 {
		t.Errorf("expected no expansions without llm, got %d", len(got))
	}
}

func TestPPTSlideNumberMatchesPowerPoint(t *testing.T) {
	// slide 1 of the PPT is the title slide, so the first content slide is slide 2
	if pptSlideNumber(1) != 2 || pptSlideNumber(12) != 13 {
		t.Fatalf("unexpected numbering: %d %d", pptSlideNumber(1), pptSlideNumber(12))
	}
}
