#!/bin/bash
set -e
# MODEL="/models/LFM 2.5-2.6B GGUF.gguf"
# URL="https://huggingface.co/LiquidAI/LFM2.5-2.6B-GGUF/resolve/main/LFM2.5-2.6B-Q8_0.gguf?download=true"

MODEL="/models/MiniCPM5 1B Claude Opus Fable5 V2 Thinking.gguf"
URL="https://huggingface.co/GnLOLot/MiniCPM5-1B-Claude-Opus-Fable5-V2-Thinking-GGUF/resolve/main/MiniCPM5-1B-Claude-Opus-Fable5-V2-Thinking-Q8_0.gguf?download=true"

if [ ! -f "$MODEL" ]; then
  echo "[llama] $MODEL not found, downloading from Hugging Face..."
  mkdir -p /models
  curl -L -o "$MODEL" "$URL"
fi

exec /app/llama-server -m "$MODEL" --port 8000 --host 0.0.0.0 -n 512 -c 8192
