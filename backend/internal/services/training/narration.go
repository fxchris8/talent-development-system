package training

import (
	"archive/zip"
	"errors"
	"fmt"
	"html"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	slideFileRe   = regexp.MustCompile(`^ppt/slides/slide(\d+)\.xml$`)
	paragraphRe   = regexp.MustCompile(`(?s)<a:p[ >].*?</a:p>`)
	textRunRe     = regexp.MustCompile(`(?s)<a:t(?: [^>]*)?>(.*?)</a:t>`)
	sentenceEndRe = regexp.MustCompile(`[.!?:;]$`)
)

// SlideNarration returns the readable text of every slide of a generated PPTX, in slide order,
// so the frontend can read it aloud with text-to-speech.
func (s *Service) SlideNarration(pptxURL string) ([]string, error) {
	if pptxURL == "" {
		return nil, errors.New("file PPT belum digenerate")
	}
	slides, err := extractPPTXSlideTexts(filepath.Join(s.outDir, filepath.Base(pptxURL)))
	if err != nil {
		return nil, fmt.Errorf("gagal membaca isi PPT: %w", err)
	}
	if len(slides) == 0 {
		return nil, errors.New("PPT tidak memiliki teks untuk dibacakan")
	}
	return slides, nil
}

// extractPPTXSlideTexts returns the readable text of each slide, in slide order. Empty slides are skipped.
func extractPPTXSlideTexts(path string) ([]string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	type slideEntry struct {
		num  int
		file *zip.File
	}
	var entries []slideEntry
	for _, f := range zr.File {
		if m := slideFileRe.FindStringSubmatch(f.Name); m != nil {
			n, _ := strconv.Atoi(m[1])
			entries = append(entries, slideEntry{num: n, file: f})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].num < entries[j].num })

	var slides []string
	for _, e := range entries {
		rc, err := e.file.Open()
		if err != nil {
			return nil, err
		}
		xmlBytes, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		if text := slideXMLToText(string(xmlBytes)); text != "" {
			slides = append(slides, text)
		}
	}
	return slides, nil
}

func slideXMLToText(xml string) string {
	var lines []string
	for _, para := range paragraphRe.FindAllString(xml, -1) {
		var sb strings.Builder
		for _, m := range textRunRe.FindAllStringSubmatch(para, -1) {
			sb.WriteString(html.UnescapeString(m[1]))
		}
		line := strings.TrimSpace(sb.String())
		if line == "" {
			continue
		}
		if !sentenceEndRe.MatchString(line) {
			line += "."
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, " ")
}
