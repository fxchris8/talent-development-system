package training

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestExtractPPTXSlideTexts(t *testing.T) {
	// buildPPTX loads public/ppt-background.png relative to the backend root
	t.Chdir("../../..")
	svc := &Service{log: logrus.New()}
	plan := &Plan{
		Title: "Dasar Kepemimpinan",
		Slides: []Slide{
			{Heading: "Pendahuluan", Bullets: []string{"Apa itu kepemimpinan", "Mengapa penting & relevan"}, SlideType: "content"},
			{Heading: "Penutup", Bullets: []string{"Terima kasih"}, SlideType: "content"},
		},
	}
	path := filepath.Join(t.TempDir(), "materi.pptx")
	if err := svc.buildPPTX(plan, path); err != nil {
		t.Fatalf("buildPPTX: %v", err)
	}

	slides, err := extractPPTXSlideTexts(path)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(slides) == 0 {
		t.Fatal("no slide text extracted")
	}
	all := strings.Join(slides, " ")
	for _, want := range []string{"Pendahuluan", "Apa itu kepemimpinan", "Mengapa penting & relevan", "Terima kasih"} {
		if !strings.Contains(all, want) {
			t.Errorf("narration is missing %q; got: %s", want, all)
		}
	}
}

func TestExtractJSON(t *testing.T) {
	cases := map[string]string{
		"```json\n{\"a\":1}\n```":        `{"a":1}`,
		"Berikut hasilnya: {\"a\":1} ok": `{"a":1}`,
		`{"a":{"b":2}}`:                  `{"a":{"b":2}}`,
	}
	for in, want := range cases {
		if got := extractJSON(in); got != want {
			t.Errorf("extractJSON(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExtractPPTXSlideTextsReturnsEverySlideInOrder(t *testing.T) {
	t.Chdir("../../..")
	svc := &Service{log: logrus.New()}
	plan := &Plan{Title: "Judul Utama"}
	for i := 1; i <= 12; i++ {
		plan.Slides = append(plan.Slides, Slide{
			Heading:   "Heading nomor " + strconv.Itoa(i),
			Bullets:   []string{"Poin pertama slide " + strconv.Itoa(i), "Poin kedua slide " + strconv.Itoa(i)},
			SlideType: "content",
		})
	}
	plan.Activities = []Activity{{Title: "Diskusi", Time: "10 menit", Instructions: "Diskusikan kasus"}}
	plan.Assessment = []QA{{Question: "Apa itu X?", Answer: "X adalah Y"}}

	path := filepath.Join(t.TempDir(), "materi.pptx")
	if err := svc.buildPPTX(plan, path); err != nil {
		t.Fatalf("buildPPTX: %v", err)
	}
	slides, err := extractPPTXSlideTexts(path)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	// title + 12 content + activities + assessment
	if want := 1 + 12 + 1 + 1; len(slides) != want {
		t.Fatalf("got %d slide narrations, want %d: %q", len(slides), want, slides)
	}
	for i := 1; i <= 12; i++ {
		got := slides[i] // slides[0] is the title slide
		if !strings.Contains(got, "Heading nomor "+strconv.Itoa(i)+".") {
			t.Errorf("slide %d out of order or missing, got %q", i, got)
		}
	}
}
