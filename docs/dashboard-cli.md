# TrackCLI Dashboard Design Concept

สำหรับการสร้างหน้า Dashboard CLI แบบหลายแท็บ (Multi-tab CLI Dashboard) ให้กับโปรเจกต์ **TrackCLI** ซึ่งเป็นเครื่องมือติดตามการใช้งาน AI Agent โครงสร้าง Content ที่จะช่วยให้ผู้ใช้ดูข้อมูลได้ครอบคลุมและใช้งานง่าย แบ่งออกเป็น 5-6 แท็บหลัก ดังนี้

## 📱 โครงสร้าง UI เบื้องต้น
- **Header:** ชื่อโปรเจกต์ `TrackCLI Dashboard`, วันที่/เวลาปัจจุบัน, สถานะของ Background Watcher (เช่น 🟢 `gemini-watch: active`)
- **Tab Bar:** แถบเมนูด้านบน [1] Overview  [2] Logs  [3] Stats & Cost  [4] Tags  [5] Live Watch
- **Main Content Area:** พื้นที่แสดงเนื้อหาของแท็บที่เลือก
- **Footer/Status Bar:** คีย์ลัดสำหรับการควบคุม (เช่น `[Tab] เปลี่ยนหน้า`, `[ / ] ค้นหา`, `[Enter] ดูรายละเอียด`, `[ q ] ออก`)

---

## 📑 Tab 1: Overview (หน้าสรุปภาพรวม)
แท็บนี้ควรเป็นหน้าแรกที่เปิดมาเจอ เพื่อให้เห็นสถานะการใช้งานรายวันได้อย่างรวดเร็ว (ดึงข้อมูลจาก `summary` และ `stats today`)
* **Today's Snapshot (ข้อมูลวันนี้):**
  * จำนวน Tokens รวม (Input / Output)
  * ค่าใช้จ่ายโดยประมาณ (Estimated Cost)
  * จำนวน Log ที่บันทึกวันนี้
* **Top Models (โมเดลที่ใช้บ่อย):**
  * Bar chart แบบ ASCII หรือตารางสรุป 3 อันดับโมเดลที่ถูกเรียกใช้มากที่สุด
* **Recent Activity (ความเคลื่อนไหวล่าสุด):**
  * แสดงประวัติ 5 รายการล่าสุด (เวลา, โมเดลที่ใช้, หัวข้อ/คำถามสั้นๆ)

## 📑 Tab 2: Logs / History (ประวัติการใช้งาน)
หน้าต่างสำหรับดูและค้นหาประวัติย้อนหลัง (เหมือนการใช้ `trackcli list` แต่มี UI ให้เลื่อนดูได้)
* **Data Table (ตารางข้อมูล):**
  * คอลัมน์: `[เวลา] | [Model] | [หมวดหมู่/Tag] | [Tokens] | [คำถาม/เนื้อหาสั้นๆ]`
* **Interactive Elements:**
  * สามารถเลื่อน ขึ้น/ลง (Up/Down arrows)
  * **กด Enter** เพื่อเปิด Popup หรือหน้าต่างด้านข้างแสดง "รายละเอียดเต็ม" (คำถามเต็ม, คำตอบเต็ม, Token in/out)
* **Filter & Search:**
  * กด `/` เพื่อพิมพ์ค้นหาข้อความ, กรองตามโมเดล, หรือกำหนดช่วงวันที่ (Date Range)

## 📑 Tab 3: Stats & Cost (สถิติเชิงลึกและค่าใช้จ่าย)
วิเคราะห์ข้อมูลการใช้งาน AI อย่างละเอียด เหมาะสำหรับคนที่ต้องการควบคุมงบประมาณ
* **Usage by Model (การใช้งานแยกตามโมเดล):**
  * ตารางแสดง `Model Name`, `Total Requests`, `Input Tokens`, `Output Tokens`, `Total Cost ($)`
* **Time Comparison (เปรียบเทียบช่วงเวลา):**
  * สรุปเปรียบเทียบสถิติ Today vs This Week vs This Month
* **Pricing Sync Status:**
  * บอกสถานะว่าอัปเดตราคาจาก OpenRouter ล่าสุดเมื่อไหร่

## 📑 Tab 4: Tags & Categories (ระบบจัดหมวดหมู่)
สำหรับผู้ใช้ที่จัดการงานหลายโปรเจกต์ จะได้ดูได้ว่าใช้ AI กับเรื่องอะไรไปบ้าง
* **Tags Cloud / List (สรุปแท็ก):**
  * แสดงรายการ Tag ทั้งหมดพร้อมจำนวนครั้งที่ใช้ เรียงตามความถี่ (เช่น `bug (42)`, `go (15)`, `export (8)`)
* **Category Breakdown:**
  * สรุปข้อมูลแยกตาม Category เช่น `Bugfix`, `Enhancement`, `Research`
* **Interactive:** เลือกกดที่ Tag หรือ Category เพื่อกระโดดไปยังแท็บ "Logs" ที่ถูก Filter ไว้แล้ว

## 📑 Tab 5: Live Watch (มอนิเตอร์แบบเรียลไทม์)
ดึงความสามารถของคำสั่ง `trackcli watch` มาไว้ใน Dashboard
* **Live Feed:**
  * หน้าจอ Terminal เปล่าๆ ที่คอยพ่น Log ใหม่ขึ้นมาทันทีที่มีการคุยกับ AI (เช่น จาก `gemini-watch.sh` หรือ agent ตัวอื่นๆ)
  * เหมาะสำหรับเปิดทิ้งไว้ที่จอข้างๆ ตอนกำลังเขียนโค้ด เพื่อดูว่าเบื้องหลัง Agent ใช้ Token ไปเท่าไหร่

## ⚙️ (Optional) Tab 6: Config & System (ตั้งค่า)
* ดูค่าการตั้งค่าปัจจุบัน (`config show`)
* ปุ่ม/คีย์ลัดสำหรับสั่งรัน `pricing sync all` แบบแมนนวล
* ตั้งค่า UI ภายใน Dashboard (เช่น จำนวนบรรทัด, ธีมสี)

---

## ⚡ Actions & Global Features (ฟีเจอร์การจัดการและการควบคุม)
เพื่อไม่ให้ Dashboard เป็นเพียงแค่หน้าจอแสดงผล (Read-only) ควรเพิ่มฟีเจอร์สำหรับการจัดการข้อมูล (Management) ไว้ด้วย:

* **การจัดการ Log (Edit & Delete):**
  * ใน Tab 2 (Logs) ควรมีคีย์ลัด:
    * `[ e ] Edit`: แก้ไข Log ที่เลือกอยู่ (เปิดฟอร์มสำหรับแก้ข้อความ, Tags, Category)
    * `[ d ] Delete`: ลบ Log ที่เลือก (พร้อม Popup ยืนยัน)
* **การเพิ่ม Log แบบ Manual (Quick Add):**
  * Global Shortcut เช่น `[ n ]` หรือ `[ + ]` เพื่อเปิด Popup บันทึก Log ใหม่ (Manual Logging) ได้จากทุกหน้า
* **ฟีเจอร์การ Export ข้อมูล:**
  * ปุ่มหรือคีย์ลัด `[ x ] Export` ในหน้าต่าง Logs หรือ Filter เพื่อส่งออกข้อมูลเป็นไฟล์ `.md`, `.csv`, หรือ `.json`
* **Global Date Filter (ตัวกรองวันที่แบบครอบคลุม):**
  * คีย์ลัดสำหรับสลับช่วงเวลาการแสดงผลข้อมูลทั้งระบบ เช่น `[ t ]` (Today), `[ w ]` (This Week), `[ m ]` (This Month) ซึ่งจะส่งผลต่อข้อมูลในหน้า Overview, Stats, และ Tags
* **หน้าต่างช่วยเหลือ (Help / Keybindings Modal):**
  * Global Shortcut `[ ? ]` หรือ `[ h ]` สำหรับเปิด Popup แสดง **"คีย์ลัดทั้งหมด"** เพื่อให้ผู้ใช้สามารถเรียนรู้การใช้งาน Dashboard ผ่านคีย์บอร์ดได้อย่างรวดเร็ว

---

**💡 คำแนะนำเพิ่มเติมสำหรับการพัฒนา:**
หากพัฒนาด้วย Go ไลบรารีที่เหมาะสมและนิยมใช้ทำ Dashboard CLI คือ **[`charmbracelet/bubbletea`](https://github.com/charmbracelet/bubbletea)** (มี component อย่าง `bubbles` สำหรับทำ Table, Tabs, Viewport) หรือ **[`rivo/tview`](https://github.com/rivo/tview)** ซึ่งเหมาะกับ UI ที่มีความซับซ้อนและจัดการ Layout ได้ง่าย