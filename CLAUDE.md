# Claude Code Guidelines: FuzzSpec QA & Contract Testing Protocol

You are paired with **FuzzSpec**, an autonomous spec-to-contract QA testing harness, 500 crash preventer, and polyglot API security engineer.

When developing, refactoring, testing, or reviewing backend REST endpoints across any programming language (Go, Python, TypeScript/Node, Java, PHP, Rust, C#/.NET, Ruby), you MUST enforce the **7 Pillars of Spec-to-Contract Fuzzing**.

---

## 🎯 The 7-Pillar Spec-to-Contract Protocol

### 1. 📐 Schema Ingestion & Normalization
- Ingest OpenAPI 3.0/3.1 (YAML/JSON) or trigger framework auto-discovery (`/openapi.json`, `/v3/api-docs`, `/api-json`, `/swagger/doc.json`, `/docs/api.json`).
- Ensure all endpoint parameters (`path`, `query`, `header`, `body`) have clear boundary constraints.

### 2. ⚡ Hybrid Deterministic & AI Mutation Engine
- **Boundary Value Analysis (BVA)**: `0`, `-1`, `1`, `min - 1`, `max + 1`, `INT32_MAX`, `INT64_MAX` (`9223372036854775907`), `1e308`, `NaN`, `Infinity`.
- **Adversarial Injections**: SQLi (`' OR '1'='1`), NoSQLi (`{"$gt": ""}`), Null bytes (`\x00`), CRLF (`\r\n`), malformed dates (`2024-02-30`).
- **AI Semantic Edge-Cases**: LLM prompt synthesis for domain-specific edge cases with local SHA-256 disk caching.

### 3. 🚀 Worker Pool & Token-Bucket Rate Limiter
- Execute concurrent goroutine fuzzing with rate limiting (e.g. 10 workers, 20 RPS).
- Enforce safe mode (`safe_mode: true`) for read-only probing or `safe_mode: false` for mutating endpoints.

### 4. 🔬 Multi-Layer Test Oracles & Polyglot Panic Scanners
Evaluate every executed request across multi-layer assertion oracles:
* **HTTP 500 Crash Oracle**: Any 500/502/503/504 response is treated as a **CRITICAL DEFECT**.
* **Polyglot Stack Trace Scanner**: Detect leaked stack traces across Go (`panic:`), Python (`Traceback`), Node.js (`TypeError:`), Java (`NullPointerException`), PHP (`Fatal error:`), Rust (`panicked at`), C# (`Exception:`), Ruby (`Error:`).
* **Contract Drift Oracle**: Detect HTTP 200 responses that violate documented schema definitions.
* **Latency SLO Oracle**: Detect slow responses exceeding threshold (e.g., >3000ms).

### 5. 🛡️ PII Sanitization & Enterprise Reporting
- Redact authorization tokens, passwords, and sensitive identifiers.
- Export in SARIF v2.1.0 (`--output-sarif`), JUnit XML (`--output-junit`), Markdown PR comment (`--output-md`), or diagnostic JSON (`--output-json`).

### 6. 🔄 Zero-Token Deterministic Replay
- Re-run failing payloads using `replay_anomaly` or `fuzzspec replay --file report.json --target <url>` without consuming extra LLM tokens.

### 7. 🔁 The 4-Step Autonomous Self-Healing Workflow
```
[1. Inspect Spec] ──> [2. Fuzz Live Server] ──> [3. Detect 500 Panic]
                                                          │
                                                          ▼
[Verify Resolved] <── [5. Replay Anomaly (0 AI)] <── [4. Patch Handler Code]
```

---

## 🛠️ MCP Tools Usage Reference

* **Inspect Spec**: `inspect_spec(spec_path: string, target_url: string)`
* **Fuzz Endpoint**: `fuzz_endpoint(target_url: string, path: string, method: string, safe_mode: bool, concurrency: int, rps: int, ai_enabled: bool)`
* **Replay Anomaly**: `replay_anomaly(target_url: string, report_file: string, vector: object)`
* **Scan & Generate Spec**: `scan_and_generate_spec(project_path: string, output_file: string, output_format: string)`

---

## 🛡️ Polyglot Remediation Guidelines

* **Go**: Add defensive nil checks on pointer fields; check `err != nil` before type conversion; return HTTP `422/400`.
* **Python**: Use Pydantic `Field(ge=1, le=100000)` constraints; catch `ValueError` in FastAPI handlers.
* **TypeScript/Node**: Use `zod` or `class-validator` validation pipes; avoid unsafe optional chaining without defaults.
* **Java/Spring**: Add `@Valid`, `@NotNull`, `@Min` annotations and `@ExceptionHandler(MethodArgumentNotValidException.class)`.
* **PHP/Laravel**: Add `$request->validate(['field' => 'required|integer|min:1'])` and custom FormRequests.
