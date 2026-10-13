"""Tiny HTTP wrapper around Piper: POST /synthesize {"slides": ["...", ...]} -> MP3."""
import io
import json
import os
import threading
import wave
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

import lameenc
from piper import PiperVoice

VOICE_NAME = os.environ.get("PIPER_VOICE", "id_ID-news_tts-medium")
VOICE = PiperVoice.load(f"/app/voices/{VOICE_NAME}.onnx")
SAMPLE_RATE = VOICE.config.sample_rate
PAUSE_BETWEEN_SLIDES_SEC = 0.8
MAX_SLIDES = 80
MAX_CHARS = 60_000

# Piper is CPU bound; one synthesis at a time keeps the container responsive.
SYNTH_LOCK = threading.Lock()


def synthesize_pcm(text: str) -> bytes:
    """Return 16-bit mono PCM for one block of text."""
    buffer = io.BytesIO()
    with wave.open(buffer, "wb") as wav_file:
        VOICE.synthesize_wav(text, wav_file)
    buffer.seek(0)
    with wave.open(buffer, "rb") as wav_file:
        return wav_file.readframes(wav_file.getnframes())


def to_mp3(pcm: bytes) -> bytes:
    encoder = lameenc.Encoder()
    encoder.set_bit_rate(48)
    encoder.set_in_sample_rate(SAMPLE_RATE)
    encoder.set_channels(1)
    encoder.set_quality(5)
    return bytes(encoder.encode(pcm)) + bytes(encoder.flush())


class Handler(BaseHTTPRequestHandler):
    def _json(self, status: int, payload: dict) -> None:
        body = json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):  # noqa: N802
        if self.path == "/health":
            self._json(200, {"status": "ok", "voice": VOICE_NAME})
        else:
            self._json(404, {"error": "not found"})

    def do_POST(self):  # noqa: N802
        if self.path != "/synthesize":
            self._json(404, {"error": "not found"})
            return
        try:
            length = int(self.headers.get("Content-Length", "0"))
            payload = json.loads(self.rfile.read(length) or b"{}")
            slides = [str(s).strip() for s in payload.get("slides", []) if str(s).strip()]
        except (ValueError, AttributeError):
            self._json(400, {"error": "invalid JSON body"})
            return
        if not slides:
            self._json(400, {"error": "slides is empty"})
            return
        if len(slides) > MAX_SLIDES or sum(len(s) for s in slides) > MAX_CHARS:
            self._json(413, {"error": "text is too long"})
            return

        pause = b"\x00\x00" * int(SAMPLE_RATE * PAUSE_BETWEEN_SLIDES_SEC)
        try:
            with SYNTH_LOCK:
                pcm = pause.join(synthesize_pcm(slide) for slide in slides)
                audio = to_mp3(pcm)
        except Exception as exc:  # report the failure instead of dropping the connection
            self._json(500, {"error": f"synthesis failed: {exc}"})
            return

        self.send_response(200)
        self.send_header("Content-Type", "audio/mpeg")
        self.send_header("Content-Length", str(len(audio)))
        self.end_headers()
        self.wfile.write(audio)

    def log_message(self, fmt, *args):  # keep container logs short
        print(f"[tts] {self.address_string()} {fmt % args}", flush=True)


if __name__ == "__main__":
    print(f"[tts] voice={VOICE_NAME} sample_rate={SAMPLE_RATE}", flush=True)
    ThreadingHTTPServer(("0.0.0.0", 5002), Handler).serve_forever()
