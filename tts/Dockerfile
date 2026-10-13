# Text-to-speech service: reads training material aloud in Indonesian with the Piper neural
# voice. Runs fully offline once built (the voice model is downloaded at build time).
FROM python:3.11-slim

WORKDIR /app

RUN pip install --no-cache-dir piper-tts==1.3.0 lameenc==1.8.1

# Indonesian voice (about 60 MB)
ARG PIPER_VOICE=id_ID-news_tts-medium
ENV PIPER_VOICE=${PIPER_VOICE}
RUN python -m piper.download_voices ${PIPER_VOICE} --data-dir /app/voices

COPY server.py .

EXPOSE 5002
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
  CMD python -c "import urllib.request;urllib.request.urlopen('http://localhost:5002/health',timeout=3)" || exit 1

CMD ["python", "server.py"]
