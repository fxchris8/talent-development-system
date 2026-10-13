package training

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	defaultTTSURL        = "http://tds_tts:5002"
	narrationAudioSuffix = "-narasi.mp3"
)

// NarrationAudioService turns the slide text of a generated PPTX into one spoken MP3 using the
// TTS container (Piper, Indonesian voice). The MP3 is stored next to the PPTX and reused.
type NarrationAudioService struct {
	log     *logrus.Logger
	http    *http.Client
	ttsURL  string
	pubBase string
	outDir  string

	mu       sync.Mutex
	inFlight map[string]*sync.Mutex // one synthesis per PPTX at a time
}

func NewNarrationAudioService(log *logrus.Logger, ttsURL, pubBase string) *NarrationAudioService {
	if ttsURL == "" {
		ttsURL = defaultTTSURL
	}
	if pubBase == "" {
		pubBase = "http://localhost:8080"
	}
	return &NarrationAudioService{
		log:      log,
		http:     &http.Client{Timeout: 15 * time.Minute},
		ttsURL:   strings.TrimRight(ttsURL, "/"),
		pubBase:  strings.TrimRight(pubBase, "/"),
		outDir:   "./public/materi",
		inFlight: make(map[string]*sync.Mutex),
	}
}

// audioName maps "materi.pptx" to "materi-narasi.mp3". The name follows the PPTX, so a
// regenerated PPTX (new file name) never plays the audio of the old one.
func audioName(pptxURL string) (pptx string, audio string, err error) {
	pptx = filepath.Base(strings.TrimSpace(pptxURL))
	if !strings.EqualFold(filepath.Ext(pptx), ".pptx") {
		return "", "", errors.New("file PPT belum digenerate")
	}
	return pptx, strings.TrimSuffix(pptx, filepath.Ext(pptx)) + narrationAudioSuffix, nil
}

func (s *NarrationAudioService) lockFor(name string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	lock, ok := s.inFlight[name]
	if !ok {
		lock = &sync.Mutex{}
		s.inFlight[name] = lock
	}
	return lock
}

// Ensure returns the public URL of the narration MP3 of a PPTX, creating it first when needed.
func (s *NarrationAudioService) Ensure(ctx context.Context, pptxURL string) (string, error) {
	pptx, audio, err := audioName(pptxURL)
	if err != nil {
		return "", err
	}
	link := fmt.Sprintf("%s/files/materi/%s", s.pubBase, audio)
	audioPath := filepath.Join(s.outDir, audio)

	lock := s.lockFor(audio)
	lock.Lock()
	defer lock.Unlock()

	if info, err := os.Stat(audioPath); err == nil && info.Size() > 0 {
		return link, nil
	}

	slides, err := extractPPTXSlideTexts(filepath.Join(s.outDir, pptx))
	if err != nil {
		return "", fmt.Errorf("gagal membaca isi PPT: %w", err)
	}
	if len(slides) == 0 {
		return "", errors.New("PPT tidak memiliki teks untuk dibacakan")
	}

	started := time.Now()
	mp3, err := s.synthesize(ctx, slides)
	if err != nil {
		return "", err
	}

	// Write to a temp file first so a half-written MP3 is never served.
	tmp := audioPath + ".tmp"
	if err := os.WriteFile(tmp, mp3, 0o644); err != nil {
		return "", fmt.Errorf("gagal menyimpan audio: %w", err)
	}
	if err := os.Rename(tmp, audioPath); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("gagal menyimpan audio: %w", err)
	}

	s.log.Infof("[TTS] %s dibuat: %d slide, %d KB, %.1f detik", audio, len(slides), len(mp3)/1024, time.Since(started).Seconds())
	return link, nil
}

// Remove deletes the narration of a PPTX that is being replaced or deleted.
func (s *NarrationAudioService) Remove(pptxURL string) {
	_, audio, err := audioName(pptxURL)
	if err != nil {
		return
	}
	if err := os.Remove(filepath.Join(s.outDir, audio)); err != nil && !os.IsNotExist(err) {
		s.log.WithError(err).Warnf("Gagal menghapus audio lama: %s", audio)
	}
}

func (s *NarrationAudioService) synthesize(ctx context.Context, slides []string) ([]byte, error) {
	payload, err := json.Marshal(map[string]any{"slides": slides})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.ttsURL+"/synthesize", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		s.log.WithError(err).Error("[TTS] layanan suara tidak bisa dihubungi")
		return nil, errors.New("layanan suara (TTS) tidak bisa dihubungi")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca audio dari layanan suara: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		s.log.Errorf("[TTS] status %d: %s", resp.StatusCode, truncate(string(body), 300))
		return nil, fmt.Errorf("layanan suara mengembalikan status %d", resp.StatusCode)
	}
	if len(body) == 0 {
		return nil, errors.New("layanan suara mengembalikan audio kosong")
	}
	return body, nil
}
