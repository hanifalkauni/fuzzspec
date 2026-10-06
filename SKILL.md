---
name: fuzzspec
description: Universal Spec-to-Contract AI Testing Harness, 500 Crash Preventer & Autonomous Self-Healing Skill for OpenAPI/REST APIs across all languages.
---

# FuzzSpec Agent Skill & Autonomous Testing Harness

Use this skill when developing, testing, or debugging backend REST APIs across any programming language (Go, Python, TypeScript/Node, Java, PHP, Rust, C#/.NET, Ruby).

## 🎯 Core Capabilities
1. **Spec-to-Contract Boundary & Adversarial Fuzzing**: Automatically mutates input schemas with Equivalence Partitioning, Boundary Value Analysis, integer overflows, null bytes, SQLi/CRLF injection, and LLM semantic edge-cases.
2. **Multi-Layer QA Oracles**: Detects 500 runtime crashes, unhandled panics, polyglot stack trace leaks (Go, Python, Node, Java, PHP, Rust, C#, Ruby), latency SLO violations, and response schema contract drift.
3. **Autonomous Self-Healing Loop**: Inspect -> Fuzz -> Diagnose Anomaly -> Patch Code -> Replay -> Verify.
4. **Zero-Token Deterministic Replay**: Validates fixes without consuming additional LLM tokens.

---

## 🔁 The Autonomous Self-Healing Workflow

```
[Inspect Endpoint Schema] ──> [Fuzz Target API] ──> [Detect 500 Crash / Anomaly]
                                                               │
                                                               ▼
[Verify Fix (Resolved)] <── [Replay Vector (0 AI)] <── [Patch Backend Handler]
```

### Step 1: Discover / Inspect Endpoints
If you need to analyze endpoint parameters and constraints:
```json
{
  "tool": "inspect_spec",
  "arguments": {
    "target_url": "http://localhost:8080"
  }
}
```

### Step 2: Run Targeted Fuzzing
Fuzz the target endpoint to surface boundary edge cases and uncaught exceptions:
```json
{
  "tool": "fuzz_endpoint",
  "arguments": {
    "target_url": "http://localhost:8080",
    "path": "/v1/orders",
    "method": "POST",
    "safe_mode": false,
    "concurrency": 5,
    "rps": 20
  }
}
```

### Step 3: Analyze Anomaly Output & cURL Reproducer
When `fuzz_endpoint` reports anomalies, review the exact finding, status code, and `curl` command:
```json
{
  "id": "vec_post_orders_003",
  "path": "/v1/orders",
  "method": "POST",
  "scenario": "Overflow Integer Quantity",
  "status_code": 500,
  "finding": "Polyglot Runtime Exception: unhandled panic integer overflow",
  "curl": "curl -X POST http://localhost:8080/v1/orders -H 'Content-Type: application/json' -d '{\"quantity\": 9223372036854775907}'"
}
```

### Step 4: Patch the Backend Code
Apply defensive validation in the handler to gracefully return standard client error codes (`400 Bad Request` or `422 Unprocessable Entity`):

* **Go (Gin/Echo):** Validate bounds with `validator.v10` or check `err != nil` before type conversion.
* **Python (FastAPI/Pydantic):** Add `Field(ge=1, le=100000)` constraints or handle `ValueError`.
* **TypeScript/Node (Express/NestJS):** Use `class-validator` / `zod` validation pipes.
* **Java (Spring Boot):** Add `@Min` / `@Max` / `@Valid` annotations or `@ExceptionHandler`.
* **PHP (Laravel):** Add `$request->validate(['quantity' => 'required|integer|min:1|max:100000'])`.

### Step 5: Verify with Replay
Re-test the specific failing payload without running the full test suite:
```json
{
  "tool": "replay_anomaly",
  "arguments": {
    "target_url": "http://localhost:8080",
    "vector": {
      "id": "vec_post_orders_003",
      "path": "/v1/orders",
      "method": "POST",
      "body": {"quantity": 9223372036854775907},
      "expected_status_family": "4xx"
    }
  }
}
```
Check that the verdict returns `"RESOLVED"` and `"all_resolved": true`.

---

## 🛠️ MCP Tools Reference

| Tool Name | Purpose | Key Arguments |
|---|---|---|
| `inspect_spec` | Ingests and details OpenAPI routes and schemas. | `spec_path` or `target_url` |
| `fuzz_endpoint` | Executes boundary & adversarial fuzzing on live server. | `target_url`, `path`, `method`, `safe_mode`, `ai_enabled` |
| `replay_anomaly` | Re-tests failing payloads to verify bug fixes. | `target_url`, `report_file` or `vector` |
| `scan_and_generate_spec` | Scaffolds an OpenAPI 3.1 contract from codebase routes. | `project_path`, `output_file`, `output_format` |

---

## ⚙️ MCP Server Configuration (`mcp_config.json`)

To enable FuzzSpec MCP in AI IDEs (Antigravity, Cursor, Claude Desktop, Windsurf, Kiro), add:

```json
{
  "mcpServers": {
    "fuzzspec": {
      "command": "npx",
      "args": [
        "-y",
        "github:hanifalkauni/fuzzspec",
        "--mcp"
      ]
    }
  }
}
```

---

## 🛡️ Remediation Cheat-Sheet for Common Anomalies

| Anomaly Type | Root Cause | Remediation Strategy |
|---|---|---|
| **Nil / Null Pointer (500)** | Accessing nested key without null-check. | Add defensive nil-check; return HTTP `422 Unprocessable Entity` with JSON validation error. |
| **Integer Overflow (500)** | Parsing values greater than `INT64_MAX` into numeric types. | Validate number string length / range before numeric conversion. Return `400 Bad Request`. |
| **SQL/NoSQL Injection (500/Leak)** | Dynamic string concatenation in SQL or query builder. | Use parameterized queries / prepared statements. Ensure raw DB error messages are not exposed. |
| **Date/UUID Parse Panic (500)** | Format parsing error on invalid dates (e.g. `2024-02-30`). | Catch parse errors in controller/DTO layer and return `400 Bad Request`. |
| **Contract Drift (200 OK)** | Response fields differ from OpenAPI specification. | Align serializer/DTO attributes to match the OpenAPI contract definitions. |
