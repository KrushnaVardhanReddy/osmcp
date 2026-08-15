#!/usr/bin/env python3
"""
Jules Batch Submitter for osmcp
Sends tasks to Jules API to create async coding sessions → GitHub PRs.

Usage:
  python3 jules_submit.py               # List all available tasks
  python3 jules_submit.py --task 1      # Submit specific task by number
  python3 jules_submit.py --list        # List available tasks
  python3 jules_submit.py --waves       # List available waves
  python3 jules_submit.py --status      # Check recent session status
  python3 jules_submit.py --file path   # Submit a custom prompt from a file
  python3 jules_submit.py --branch feat # Target a specific branch
  python3 jules_submit.py --wave e2e    # Submit a predefined wave of tasks
"""

import json
import urllib.request
import sys
import os

# ──────────────────────────────────────────────────────────────────────────────
# Config — loads API key from .env.local or .env (never hardcode secrets)
# ──────────────────────────────────────────────────────────────────────────────

def _load_api_key():
    """Read JULES_API_KEY from environment, .env.local, or .env."""
    key = os.environ.get("JULES_API_KEY")
    if key:
        return key
    for envfile in [".env.local", ".env"]:
        # Look in the repo root, not the scripts/ directory
        repo_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
        path = os.path.join(repo_root, envfile)
        if os.path.exists(path):
            with open(path) as f:
                for line in f:
                    line = line.strip()
                    if line.startswith("JULES_API_KEY="):
                        return line.split("=", 1)[1].strip()
    print("❌ JULES_API_KEY not found in environment, .env.local, or .env")
    sys.exit(1)

API_KEY = _load_api_key()
API_URL = "https://jules.googleapis.com/v1alpha/sessions"

# ── Repo root (one level up from scripts/) ────────────────────────────────────
REPO_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# ── Update this once the GitHub repo is created ──────────────────────────────
REPO_SOURCE = "sources/github/KrushnaVardhanReddy/osmcp"

# Parse branch from args if provided
BRANCH = "feature/dev"
if "--branch" in sys.argv:
    idx = sys.argv.index("--branch")
    if idx + 1 < len(sys.argv):
        BRANCH = sys.argv[idx + 1]

# ──────────────────────────────────────────────────────────────────────────────
# Mandatory safety rules (prepended to every prompt)
# ──────────────────────────────────────────────────────────────────────────────

SAFETY_RULES = """
MANDATORY RULES — VIOLATION = REJECTED PR:
1. NEVER stub, mock, or TODO existing implementation code.
2. Every file you modify MUST still build — run `go build ./...` before committing.
3. If a test fails, FIX the code or test — do NOT delete or skip tests.
4. Do NOT alter any file in docs/specs/ — those are the source of truth. Implement from them, never rewrite them.
5. All executions must respect policy limits.
6. UNIT TESTS ARE MANDATORY: Every feature PR must include `*_test.go` files. Do not submit code without tests.
7. Commit message must start with "jules: " prefix.
8. 100% SPEC-FIRST RULE: If your implementation deviates from the spec in docs/specs/, STOP and flag it.
9. WIKI MAINTAINER: Read `docs/wiki/Home.md` first to understand project architecture. Update the wiki if your implementation introduces new concepts.

Project: osmcp — A typed, policy-controlled OS capability layer for AI agents.
Tech stack:
- Language: Go 1.22+
- Core Dependency: modelcontextprotocol/go-sdk
- Config: TOML (BurntSushi/toml)
- Tests: Go testing (unit and e2e)

Repo layout:
  cmd/osmcp/      — Entry point for the MCP server binary
  internal/       — Core implementation (policy, executor, tools, response, audit)
  e2e/            — End-to-end tests calling the compiled binary over stdio
  docs/specs/     — Source-of-truth specification files (READ ONLY for Jules)
  docs/tasks/     — Actionable Jules implementation tasks
  docs/wiki/      — High-level architectural overviews and philosophy
  scripts/        — Automation scripts (jules_submit.py)
""".strip()

# ──────────────────────────────────────────────────────────────────────────────
# Task definitions — populate as implementation progresses
# ──────────────────────────────────────────────────────────────────────────────

def _load_prompt(relative_path):
    """Load a prompt file lazily — returns its content or an error string."""
    full_path = os.path.join(REPO_ROOT, relative_path)
    if not os.path.exists(full_path):
        return f"ERROR: Prompt file not found: {full_path}"
    with open(full_path) as f:
        return f.read()


TASKS = {
    1: {
        "name": "Task 1 — Foundation",
        "phase": "phase-1",
        "prompt": _load_prompt("docs/tasks/phase-1/task1_foundation.md"),
    },
    2: {
        "name": "Task 2 — grep and Server",
        "phase": "phase-1",
        "prompt": _load_prompt("docs/tasks/phase-1/task2_grep_and_server.md"),
    },
    4: {
        "name": "Task 4 — File Inspection (ls, cat, stat, wc)",
        "phase": "phase-1",
        "prompt": _load_prompt("docs/tasks/phase-1/task4_file_inspection.md"),
    },
    5: {
        "name": "Task 5 — Git Intelligence (git_status, git_diff, git_log)",
        "phase": "phase-1",
        "prompt": _load_prompt("docs/tasks/phase-1/task5_git_tools.md"),
    },
    6: {
        "name": "Task 6 — find Tool",
        "phase": "phase-1",
        "prompt": _load_prompt("docs/tasks/phase-1/task6_find.md"),
    },
    7: {
        "name": "Task 7 — Transform Tools (jq, sed, diff)",
        "phase": "phase-1",
        "prompt": _load_prompt("docs/tasks/phase-1/task7_transform_tools.md"),
    },
    8: {
        "name": "Task 8 — Utility Tools (tree, head, tail, du)",
        "phase": "phase-1",
        "prompt": _load_prompt("docs/tasks/phase-1/task8_utility_tools.md"),
    },
    9: {
        "name": "Task 9 — E2E Testing",
        "phase": "phase-1",
        "prompt": _load_prompt("docs/tasks/phase-1/task9_e2e_testing.md"),
    },
    11: {
        "name": "Task 11 — File System Mutation",
        "phase": "phase-2",
        "prompt": _load_prompt("docs/tasks/phase-2/task11_fs_mutation.md"),
    },
    12: {
        "name": "Task 12 — Git Mutation",
        "phase": "phase-2",
        "prompt": _load_prompt("docs/tasks/phase-2/task12_git_mutation.md"),
    },
    13: {
        "name": "Task 13 — Patch Mutation",
        "phase": "phase-2",
        "prompt": _load_prompt("docs/tasks/phase-2/task13_patch_mutation.md"),
    },
    14: {
        "name": "Task 14 — Phase 2 E2E Testing",
        "phase": "phase-2",
        "prompt": _load_prompt("docs/tasks/phase-2/task14_e2e_testing.md"),
    },
    17: {
        "name": "Task 17 — sort",
        "phase": "phase-2",
        "prompt": _load_prompt("docs/tasks/phase-2/task17_sort.md"),
    },
    18: {
        "name": "Task 18 — awk",
        "phase": "phase-2",
        "prompt": _load_prompt("docs/tasks/phase-2/task18_awk.md"),
    },
    19: {
        "name": "Task 19 — tar",
        "phase": "phase-2",
        "prompt": _load_prompt("docs/tasks/phase-2/task19_tar.md"),
    },
    15: {
        "name": "Task 15 — Templates & Agent Onboarding",
        "phase": "phase-3",
        "prompt": _load_prompt("docs/tasks/phase-3/task15_templates_onboarding.md"),
    },
    16: {
        "name": "Task 16 — run_script (Tier 2 Execution Engine)",
        "phase": "phase-3",
        "prompt": _load_prompt("docs/tasks/phase-3/task16_run_script.md"),
    },
}

# ── Wave definitions — groups of task IDs to submit together ─────────────────
WAVES = {
    "phase-1": [4, 5, 6, 7, 8, 9],
    "phase-2": [11, 12, 13, 14, 17, 18, 19],
    "phase-3": [15, 16],
}

# ──────────────────────────────────────────────────────────────────────────────
# Submission logic
# ──────────────────────────────────────────────────────────────────────────────

def submit_task(task_num):
    if task_num not in TASKS:
        print(f"❌ Task {task_num} not found. Use --list to see available tasks.")
        sys.exit(1)

    task = TASKS[task_num]
    full_prompt = SAFETY_RULES + "\n\n---\n\n" + task["prompt"]

    payload = json.dumps({
        "prompt": full_prompt,
        "sourceContext": {
            "source": REPO_SOURCE,
            "githubRepoContext": {
                "startingBranch": BRANCH
            }
        }
    }).encode()

    req = urllib.request.Request(
        API_URL,
        data=payload,
        headers={
            "Content-Type": "application/json",
            "x-goog-api-key": API_KEY
        },
        method="POST"
    )

    print(f"🚀 Submitting: [{task_num}] {task['name']} → branch: {BRANCH}")
    try:
        with urllib.request.urlopen(req) as resp:
            result = json.loads(resp.read())
            session_id = result.get("name", "unknown").split("/")[-1]
            print(f"✅ Session created: {session_id}")
            print(f"   View at: https://jules.google.com/session/{session_id}")
    except urllib.error.HTTPError as e:
        print(f"❌ HTTP {e.code}: {e.read().decode()}")
        sys.exit(1)


def submit_wave(wave_name):
    if wave_name not in WAVES:
        print(f"❌ Wave '{wave_name}' not found.")
        print(f"   Available waves: {', '.join(WAVES.keys())}")
        sys.exit(1)
    task_ids = WAVES[wave_name]
    print(f"\n🌊 Submitting wave '{wave_name}' — {len(task_ids)} task(s):\n")
    for task_id in task_ids:
        if task_id in TASKS:
            print(f"  → [{task_id}] {TASKS[task_id]['name']}")
        else:
            print(f"  ⚠️  Task {task_id} not found in TASKS dict — skipping")
    print()
    confirm = input("Confirm submission? (y/N): ").strip().lower()
    if confirm != "y":
        print("Aborted.")
        sys.exit(0)
    print()
    for task_id in task_ids:
        if task_id in TASKS:
            submit_task(task_id)
    print(f"\n✅ Wave '{wave_name}' submitted — {len(task_ids)} session(s) created.")


def submit_file(filepath):
    if not os.path.exists(filepath):
        print(f"❌ File not found: {filepath}")
        sys.exit(1)
    with open(filepath) as f:
        prompt_content = f.read()

    full_prompt = SAFETY_RULES + "\n\n---\n\n" + prompt_content
    payload = json.dumps({
        "prompt": full_prompt,
        "sourceContext": {
            "source": REPO_SOURCE,
            "githubRepoContext": {
                "startingBranch": BRANCH
            }
        }
    }).encode()

    req = urllib.request.Request(
        API_URL,
        data=payload,
        headers={
            "Content-Type": "application/json",
            "x-goog-api-key": API_KEY
        },
        method="POST"
    )

    print(f"🚀 Submitting custom prompt from: {filepath} → branch: {BRANCH}")
    try:
        with urllib.request.urlopen(req) as resp:
            result = json.loads(resp.read())
            session_id = result.get("name", "unknown").split("/")[-1]
            print(f"✅ Session created: {session_id}")
    except urllib.error.HTTPError as e:
        print(f"❌ HTTP {e.code}: {e.read().decode()}")
        sys.exit(1)


def list_tasks():
    print("\n📋 Available osmcp Jules Tasks:\n")
    for num, task in sorted(TASKS.items()):
        wave_tag = f"  [wave: {task['wave']}]" if "wave" in task else ""
        print(f"  [{num:>3}] {task['name']}  ({task['phase']}){wave_tag}")
    print()


def list_waves():
    print("\n🌊 Available Jules Waves:\n")
    for wave_name, task_ids in WAVES.items():
        names = [TASKS[t]["name"] for t in task_ids if t in TASKS]
        print(f"  {wave_name}:")
        for n in names:
            print(f"    • {n}")
    print()
    print("Submit a wave with: python3 jules_submit.py --wave <wave-name>\n")


def main():
    args = sys.argv[1:]

    if not args or "--help" in args or "-h" in args:
        print(__doc__)
        sys.exit(0)

    if "--list" in args:
        list_tasks()
        sys.exit(0)

    if "--waves" in args:
        list_waves()
        sys.exit(0)

    if "--file" in args:
        idx = args.index("--file")
        if idx + 1 >= len(args):
            print("❌ Please specify a file path after --file.")
            sys.exit(1)
        submit_file(args[idx + 1])
        sys.exit(0)

    if "--wave" in args:
        idx = args.index("--wave")
        if idx + 1 >= len(args):
            print("❌ Please specify a wave name after --wave.")
            print(f"   Available: {', '.join(WAVES.keys())}")
            sys.exit(1)
        submit_wave(args[idx + 1])
        sys.exit(0)

    if "--task" in args:
        idx = args.index("--task")
        if idx + 1 >= len(args):
            print("❌ Please specify a task number after --task.")
            sys.exit(1)
        submit_task(int(args[idx + 1]))
        sys.exit(0)

    # Default: list tasks
    list_tasks()


if __name__ == "__main__":
    main()
