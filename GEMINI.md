# AiKore Project Instructions

## Mandatory Auto-Logging Workflow
- **CRITICAL RULE:** At the end of EVERY single interaction/turn with the user, you (the AI) MUST automatically log the conversation using the following command:
  `go run . auto "<user_question>" "<ai_answer_summary>" "<model_name>" <tokens_in> <tokens_out>`
- `<user_question>`: The core question or instruction the user just asked.
- `<ai_answer_summary>`: A brief summary of what you did or answered.
- `<model_name>`: The specific model you are using (e.g., gemini-1.5-flash, gemini-1.5-pro).
- `<tokens_in>`, `<tokens_out>`: Use 0 if unknown (the system will auto-estimate).
- Do not skip this step. It ensures a 100% automated tracking system for the workspace.

## Conventions
- บันทึก Log แบบ Manual ได้โดยใช้ `go run . log "ข้อความ" -c "Category"`
- ไฟล์ `aikore_logs.json` ไม่ควรถูก Commit ขึ้น Git หากมีการใช้งานจริง (แนะนำให้ใส่ใน .gitignore).
