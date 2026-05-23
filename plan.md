# แผนการทำ Auto-Config สำหรับ Global Rules

เราจะสร้างฟังก์ชันให้ระบบสามารถดัดแปลงไฟล์ตั้งค่า (Global Settings) ของเครื่องมือ AI แต่ละตัวได้โดยตรง เพื่อช่วยให้ผู้ใช้ไม่ต้องนำคำสั่งไปแปะด้วยตัวเอง (ยกเว้นแอปที่ไม่อนุญาต)

```text
[ User runs: atrack init --global ]
                 │
                 ▼
 { AgentTrack Auto-Config Engine }
                 │
                 ├────────────────────────────────────────────────────────┐
                 │                                                        │
                 ▼                                                        ▼
      === Automated Setup (Auto-Config) ===                      === Manual Setup ===
                 │                                                        │
 ┌───────────┬───┴───────┬────────────┬────────────┬─────────────┐        ▼
 ▼           ▼           ▼            ▼            ▼             ▼     [Cursor]
[Claude]  [Aider]   [Cline/Roo]  [Windsurf]   [Antigravity] [Open Interp.]┊
[Code  ]                         [Continue]   [Shell-GPT  ]               ┊ Cannot Auto-Config
 │           │           │            │            │             │        ┊ Cloud Data
 │ Run cmd   │ Append to │ Inject to  │ Write JSON │ Write role  │ Write  ▼
 │           │ conf.yml  │ .json      │ & rules    │ & system_p. │ .yaml[Show Manual ]
 ▼           ▼           ▼            ▼            ▼             ▼      [Instructions]
(OK)        (OK)        (OK)         (OK)         (OK)          (OK)      ┊
                                                                          ┊ User Pastes
                                                                          ▼
                                                                        (Done)
```

## สิ่งที่ Auto-Config ทำได้อัตโนมัติ

1. **Claude Code:** 
   - ให้รันคำสั่ง `claude config set --global customInstructions ...` อัตโนมัติผ่านเบื้องหลัง
2. **Aider:** 
   - ให้เปิดไฟล์ `~/.aider.conf.yml` แล้วเขียนค่า `message: | ...` ต่อท้ายให้เลย
3. **Cline และ Roo Code:** 
   - ให้เปิดอ่านไฟล์ `settings.json` ในโฟลเดอร์ `globalStorage` ของ VS Code Extension เพื่อแก้ค่า `"customInstructions"`
4. **Windsurf:** 
   - สร้างไฟล์ `~/.windsurfrules` ให้ที่โฟลเดอร์หลักของผู้ใช้
5. **Continue.dev (New!):**
   - เข้าไปอ่านไฟล์ `~/.continue/config.json` และฝังคำสั่งลงใน `"systemMessage"`
6. **Open Interpreter (New!):**
   - สร้างและอัปเดตไฟล์โปรไฟล์ที่ `~/.config/open-interpreter/profiles/default.yaml`
7. **Shell-GPT / sgpt (New!):**
   - สร้าง Role พื้นฐานของระบบลงในโฟลเดอร์ `~/.config/shell_gpt/roles/default.json`
8. **Antigravity (Gemini CLI):**
   - นำไฟล์ไปวางใน `~/.gemini/config/skills/AgentTrack.md` หรือ `system_prompt.txt`

## สิ่งที่ทำไม่ได้ (ต้องทำมือ)

> [!WARNING]
> **Cursor**
> ตัวแอป Cursor เก็บค่า "Rules for AI" (Global Rules) ไว้บนคลาวด์/เซิร์ฟเวอร์ของบัญชีผู้ใช้ หรือเก็บในฐานข้อมูลที่เข้ารหัส ไม่ได้บันทึกเป็นไฟล์ Text ธรรมดาเหมือนเครื่องมืออื่น 
> ดังนั้น **เราไม่สามารถ Auto-Config ให้ Cursor ได้** ระบบจะต้องพิมพ์แจ้งเตือนให้คุณไปก๊อปปี้วางในตั้งค่าของ Cursor ด้วยตัวเอง 1 ครั้งครับ
