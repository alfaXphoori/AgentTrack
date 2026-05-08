import os
import json
import subprocess
import hashlib
import time
import glob

# Configuration
STORAGE_PATH = os.path.expanduser("~/Library/Application Support/Code/User/workspaceStorage")
STATE_DIR = os.path.expanduser("~/.atrack/vscode_copilot_state")
ATRACK_BIN = "/Users/phoori/go/bin/atrack"

if not os.path.exists(STATE_DIR):
    os.makedirs(STATE_DIR)

def get_logged_count(session_id):
    state_file = os.path.join(STATE_DIR, f"{session_id}.logged")
    if os.path.exists(state_file):
        try:
            with open(state_file, "r") as f:
                return int(f.read().strip())
        except:
            return 0
    return 0

def save_logged_count(session_id, count):
    state_file = os.path.join(STATE_DIR, f"{session_id}.logged")
    with open(state_file, "w") as f:
        f.write(str(count))

def extract_response_text(data_list):
    """Deeply extract text from various VS Code Copilot response structures"""
    texts = []
    for item in data_list:
        if isinstance(item, dict):
            # Format 1: Direct value
            if "value" in item and isinstance(item["value"], str):
                texts.append(item["value"])
            # Format 2: Content parts
            elif "content" in item and isinstance(item["content"], str):
                texts.append(item["content"])
            # Format 3: Nested message text
            elif "message" in item and isinstance(item["message"], dict):
                msg_text = item["message"].get("text", "")
                if msg_text:
                    texts.append(msg_text)
    return "\n".join(texts) if texts else ""

def process_file(file_path):
    try:
        with open(file_path, "r") as f:
            lines = f.readlines()
    except Exception:
        return

    session_id = None
    requests = []
    
    # First pass: Get Session ID and all requests
    for line in lines:
        try:
            data = json.loads(line)
            # Session Init
            if data.get("kind") == 0:
                v = data.get("v", {})
                if not session_id:
                    session_id = v.get("sessionId")
                
                reqs = v.get("requests", [])
                for r in reqs:
                    req_id = r.get("requestId")
                    if req_id and not any(x["requestId"] == req_id for x in requests):
                        requests.append(r)
            
            # Incremental updates
            if data.get("kind") == 2:
                v = data.get("v", [])
                k = data.get("k", [])
                if k == ["requests"] and isinstance(v, list):
                    for r in v:
                        req_id = r.get("requestId")
                        if req_id and not any(x["requestId"] == req_id for x in requests):
                            requests.append(r)
        except Exception:
            continue

    if not session_id or not requests:
        return

    logged_count = get_logged_count(session_id)
    if len(requests) <= logged_count:
        return

    # Second pass: Find final responses for the requests
    # The JSONL format often has kind:1 messages that update request results
    for line in lines:
        try:
            data = json.loads(line)
            if data.get("kind") == 2:
                v = data.get("v", [])
                k = data.get("k", [])
                # If updating a specific request response
                # Format: k = ["requests", index, "response"]
                if len(k) >= 3 and k[0] == "requests" and k[2] == "response":
                    idx = k[1]
                    if idx < len(requests):
                        requests[idx]["response"] = v
        except:
            continue

    # Log only NEW pairs
    new_requests = requests[logged_count:]
    for req in new_requests:
        prompt = req.get("message", {}).get("text", "")
        if not prompt:
            continue
            
        model = req.get("modelId", "vscode-copilot")
        
        # Extract response
        response_text = extract_response_text(req.get("response", []))
        if not response_text:
            response_text = "AI Response (Content hidden or pending)"

        # Log to atrack
        cmd = [
            ATRACK_BIN, "auto",
            prompt,
            response_text[:100], # Summary
            model,
            "0", "0", "0",
            session_id,
            "success",
            "",
            "vscode-copilot,auto"
        ]
        subprocess.run(cmd, capture_output=True)
        print(f"✅ Logged VS Code Copilot: {prompt[:50]}...")

    save_logged_count(session_id, len(requests))

def main():
    print("🔍 Starting VS Code Copilot Watcher v2 (Fixed Answers)...")
    while True:
        pattern = os.path.join(STORAGE_PATH, "*/chatSessions/*.jsonl")
        files = glob.glob(pattern)
        for f in files:
            process_file(f)
        time.sleep(5)

if __name__ == "__main__":
    main()
