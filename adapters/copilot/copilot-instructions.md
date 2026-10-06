# GitHub Copilot Custom Instructions: FuzzSpec QA & Contract Testing

You are paired with **FuzzSpec**, an autonomous spec-to-contract QA testing harness, 500 crash preventer, and polyglot API security engineer.

When writing, reviewing, or refactoring backend REST APIs across any programming language (Go, Python, TypeScript/Node, Java, PHP, Rust, C#/.NET, Ruby), you MUST adhere to the following **FuzzSpec Testing Guidelines**:

---

## 🎯 The 7-Pillar Spec-to-Contract Guidelines

### 1. 📐 Schema Validation & Boundary Enforcement
- Strictly validate all input parameters (`path`, `query`, `header`, `body`) against schema constraints.
- Guard against Boundary Value Analysis (BVA) anomalies: `0`, `-1`, `1`, `INT32_MAX`, `INT64_MAX` (`9223372036854775907`), empty strings `""`, 10KB payloads.

### 2. ⚡ Adversarial Injection Prevention
- Enforce defenses against SQL injection (`' OR '1'='1`), NoSQL injection (`{"$gt": ""}`), Null bytes (`\x00`), and CRLF (`\r\n`).
- Never concatenate raw user input into database queries or system commands.

### 3. 🔬 Zero Unhandled 500 Crashes
- Catch all runtime type conversions, parsing errors, and nil/null references before they cause uncaught server panics.
- Return structured client errors (`400 Bad Request` or `422 Unprocessable Entity`) instead of HTTP 500.

### 4. 🛡️ Polyglot Stack Trace Leakage Prevention
- Ensure production error handlers do not leak internal stack traces or database error messages across any runtime (Go, Python, Node, Java, PHP, Rust, C#, Ruby).

### 5. 📜 Response Schema Contract Compliance
- Ensure returned JSON objects strictly conform to the OpenAPI 3.0/3.1 documented response contracts to prevent client breakage (*Contract Drift*).

### 6. 🔄 Zero-Token Deterministic Replay
- Use `fuzzspec replay --file report.json --target <url>` or the `replay_anomaly` MCP tool to verify that bug fixes resolve previous defects with zero additional AI token cost.

### 7. 🔁 Autonomous Self-Healing Protocol
```
[Inspect Endpoint Contract] ──> [Fuzz Live API] ──> [Isolate Anomaly / 500 Panic]
                                                              │
                                                              ▼
[Confirm Resolution] <── [Replay Payload (0 AI)] <── [Patch Defensive Validation]
```

---

## 🛠️ MCP Tools Overview

* **`inspect_spec`**: Inspects OpenAPI routes, parameter types, and response schemas.
* **`fuzz_endpoint`**: Executes concurrent boundary and adversarial mutations against a live server.
* **`replay_anomaly`**: Deterministically verifies bug fix resolution with 0 AI cost.
* **`scan_and_generate_spec`**: Generates baseline OpenAPI 3.1 specifications from codebase routes.

---

## 🛡️ Remediation Quick Reference

* **Go**: Validate pointer existence (`if item == nil`); use `strconv.ParseInt` with bounds checking.
* **Python/FastAPI**: Use Pydantic `Field(ge=1, le=100000)` and custom exception handlers.
* **Node/Express**: Use `zod` schema parsers; handle undefined body fields gracefully.
* **Java/Spring**: Add `@Valid`, `@NotNull`, and `@ControllerAdvice` global exception handlers.
* **PHP/Laravel**: Use FormRequests with strict validation rules.
