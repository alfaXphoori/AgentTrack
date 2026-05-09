#!/bin/bash
# gemiatrack - Gemini CLI wrapper with AgentTrack auto-logging

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ATRACK_BIN="$(command -v atrack || echo "$SCRIPT_DIR/../atrack")"

# Auto-detect live model from latest Gemini CLI session file
detect_live_model() {
  "$ATRACK_BIN" internal-detect-gemini
}

LIVE_MODEL=$(detect_live_model 2>/dev/null)
MODEL="${GEMINI_MODEL:-${LIVE_MODEL:-gemini-2.5-flash}}"

echo ""
printf "\033[1;32m╔══════════════════════════════════════════╗\033[0m\n"
printf "\033[1;32m║  Gemini CLI  +  AgentTrack Auto-Logger   ║\033[0m\n"
printf "\033[1;32m║  Model: %-33s║\033[0m\n" "$MODEL"
printf "\033[1;32m║  Type 'exit' or '/model' to switch       ║\033[0m\n"
printf "\033[1;32m╚══════════════════════════════════════════╝\033[0m\n"
echo ""

while true; do
  printf "\033[1;34mYou:\033[0m "
  read -r QUESTION
  
  [ -z "$QUESTION" ] && continue
  [[ "$QUESTION" == "exit" || "$QUESTION" == "quit" ]] && echo "Bye!" && break
  
  if [ "$QUESTION" = "/model" ]; then
    printf "Switch model (current: \033[1m$MODEL\033[0m): "
    read -r NEW_MODEL
    [ -n "$NEW_MODEL" ] && MODEL="$NEW_MODEL"
    printf "✅ Model set to: \033[1m$MODEL\033[0m\n\n"
    continue
  fi
  
  echo ""
  printf "\033[1;33mGemini ($MODEL):\033[0m\n"
  printf "─────────────────────────────────────────\n"
  
  ANSWER=$(gemini -m "$MODEL" -p "$QUESTION" 2>/dev/null)
  echo "$ANSWER"
  echo ""
  
  TOKENS_IN=$(( $(echo -n "$QUESTION" | wc -c | tr -d ' ') / 4 + 1 ))
  TOKENS_OUT=$(( $(echo -n "$ANSWER" | wc -c | tr -d ' ') / 4 + 1 ))
  ANSWER_SUMMARY=$(echo "$ANSWER" | head -1 | cut -c1-80)
  
  "$ATRACK_BIN" auto "$QUESTION" "$ANSWER_SUMMARY" "$MODEL" $TOKENS_IN $TOKENS_OUT 2>/dev/null
  
  printf "📋 \033[2mLogged → model: $MODEL | ~$TOKENS_IN in | ~$TOKENS_OUT out\033[0m\n\n"
done
