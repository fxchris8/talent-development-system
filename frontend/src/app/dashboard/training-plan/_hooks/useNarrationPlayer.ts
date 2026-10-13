import { useCallback, useEffect, useRef, useState } from "react";
import { authFetch } from "../../../lib/auth-fetch";

export interface NarrationProgress {
  current: number; // detik yang sudah diputar
  duration: number; // total detik, 0 selama belum diketahui
}

interface NarrationPlayer {
  playingNo: number | null;
  loadingNo: number | null;
  progress: NarrationProgress | null;
  toggle: (no: number, pptxUrl: string) => Promise<void>;
  stop: () => void;
}

// Plays the Indonesian narration of a generated PPT. The audio is made by the server (neural
// text-to-speech), so it sounds the same on every device and needs no voice installed.
export function useNarrationPlayer(
  apiUrl: string,
  onError: (message: string) => void
): NarrationPlayer {
  const [playingNo, setPlayingNo] = useState<number | null>(null);
  const [loadingNo, setLoadingNo] = useState<number | null>(null);
  const [progress, setProgress] = useState<NarrationProgress | null>(null);
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const runRef = useRef(0); // bumps on every start/stop so a slow request cannot start late

  const stop = useCallback(() => {
    runRef.current += 1;
    const audio = audioRef.current;
    if (audio) {
      audioRef.current = null;
      audio.pause();
      audio.removeAttribute("src");
      audio.load();
    }
    setPlayingNo(null);
    setLoadingNo(null);
    setProgress(null);
  }, []);

  // Leaving the page (or this tab) stops the sound.
  useEffect(() => stop, [stop]);

  const toggle = useCallback(
    async (no: number, pptxUrl: string) => {
      if (playingNo === no || loadingNo === no) {
        stop();
        return;
      }

      stop();
      const run = runRef.current;
      setLoadingNo(no);
      try {
        const res = await authFetch(`${apiUrl}/trainings/narration-audio`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ pptx_url: pptxUrl }),
        });
        const result = await res.json();
        if (run !== runRef.current) return; // stopped or switched while the audio was made
        if (!res.ok || !result.audio_url) {
          onError(result.error || "Gagal menyiapkan suara materi.");
          setLoadingNo(null);
          return;
        }

        const audio = new Audio(result.audio_url as string);
        audioRef.current = audio;
        audio.ontimeupdate = () => {
          if (audioRef.current !== audio) return;
          setProgress({ current: audio.currentTime, duration: audio.duration || 0 });
        };
        audio.onended = () => {
          if (audioRef.current === audio) stop();
        };
        audio.onerror = () => {
          if (audioRef.current !== audio) return;
          stop();
          onError("Gagal memutar suara materi.");
        };

        await audio.play();
        if (run !== runRef.current) return;
        setLoadingNo(null);
        setPlayingNo(no);
      } catch (error) {
        if (run !== runRef.current) return;
        stop();
        // Browsers refuse to start audio long after the click that asked for it.
        const blocked = error instanceof DOMException && error.name === "NotAllowedError";
        onError(
          blocked
            ? "Suara sudah siap. Klik Dengarkan sekali lagi untuk memutarnya."
            : "Terjadi kesalahan saat menyiapkan suara materi."
        );
      }
    },
    [apiUrl, loadingNo, onError, playingNo, stop]
  );

  return { playingNo, loadingNo, progress, toggle, stop };
}
