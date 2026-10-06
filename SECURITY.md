# Security Policy & Threat Model

At **FuzzSpec**, security is a foundational principle. Because FuzzSpec interacts with OpenAPI specifications, executes live HTTP fuzzing, and interfaces with AI coding agents (Cursor, Claude, Copilot, Antigravity, Windsurf), we enforce a **Zero-Trust Security & Defense-in-Depth** architecture to protect your credentials, infrastructure, and downstream services.

---

## 🛡️ Supported Versions

We actively provide security patches and bug fixes for the following versions:

| Version | Supported | Notes |
| :--- | :---: | :--- |
| `v1.x.x` | ✅ Yes | Latest active release line |
| `< 1.0.0` | ❌ No | Pre-release beta builds |

---

## 🔒 5-Layer Zero-Trust Credential Protection

FuzzSpec implements 5 distinct layers of security to ensure API tokens, database URIs, private keys, and environment variables are **never leaked** into AI chat logs, prompt contexts, CI/CD comments, or public repositories:

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                    5-LAYER CREDENTIAL PROTECTION PIPELINE                       │
├─────────────────────────────────────────────────────────────────────────────────┤
│ 1. Agent Boundary   : .cursorignore & .claudeignore isolate .env & secrets      │
│ 2. Scanner Blacklist: Parser ignores .git, .aws, .ssh, *.pem, credentials.json  │
│ 3. RAM-Only Auth    : token_env dynamically resolves keys into OS RAM only     │
│ 4. Stream Redactor  : Auto-sanitizes Bearer tokens, API keys, DB URIs to REDACTED │
│ 5. Prompt Scrubbing : LLM only receives schema types, NEVER raw auth or tokens   │
└─────────────────────────────────────────────────────────────────────────────────┘
```

1. **Agent Context Isolation (`.cursorignore`, `.claudeignore`, `.gitignore`):**  
   Files matching `.env*`, `credentials.json`, `*.pem`, `*.key`, `id_rsa`, and `.aws` are excluded from agent indexing, search embeddings, and IDE context windows.
2. **Engine-Level Scanner Blacklist (`tools_scan.go`, `discover.go`):**  
   The built-in OpenAPI auto-discovery and route discovery scanners explicitly skip secret files and sensitive directories.
3. **RAM-Only Dynamic Secret Injection (`token_env`):**  
   FuzzSpec forbids hardcoding raw secret tokens in configuration files (`fuzzspec.yaml`). Tokens are referenced via environment variable keys (e.g., `token_env: "AUTH_TOKEN"`) and read directly from OS memory at HTTP request dispatch time.
4. **Real-Time Stream Redactor (`sanitizer.go`):**  
   All terminal outputs, MCP tool responses, SARIF v2.1.0 exports, JUnit XML reports, GitHub PR comments, and cURL reproducers are parsed through a multi-regex redactor that masks Bearer tokens, OpenAI/Anthropic/AWS/GitHub API keys, DB connection strings, and passwords into `[REDACTED]`.
5. **LLM Prompt Scrubbing (`ai_generator.go`):**  
   When generating boundary and adversarial test cases with LLMs (Gemini, OpenAI, Claude), FuzzSpec transmits **only the OpenAPI schema shapes** (field names, types, enums, format constraints). No authorization headers, live session cookies, or database records are sent to AI providers.

---

## ⚠️ Threat Model & Operational Risks

When performing automated API contract testing and adversarial fuzzing against running web services, be aware of the following operational risks and their mitigations:

### 1. Accidental State Mutation & Data Loss
* **Risk:** Automated fuzzing of state-changing HTTP methods (`POST`, `PUT`, `PATCH`, `DELETE`) with boundary IDs or adversarial payloads could modify or delete existing records.
* **Mitigation:**
  * **Safe Mode:** Use `--safe-mode=true` to restrict fuzzing strictly to idempotent/read-only HTTP methods (`GET`, `HEAD`, `OPTIONS`).
  * **Isolated Environment:** Always run FuzzSpec against local containers, staging instances, or isolated test databases seeded with disposable data. **Never run destructive fuzzing directly on production databases.**

### 2. Denial of Service (DoS) & Resource Exhaustion
* **Risk:** High concurrency, rapid request rates, or extreme boundary payloads (e.g., INT64 overflows, deeply nested JSON, ReDoS strings) could exhaust CPU, memory (OOM), or database connection pools on the target service.
* **Mitigation:**
  * **Rate Limiting:** Regulate throughput using `--rps <N>` (token bucket algorithm, default 50 RPS).
  * **Bounded Concurrency:** Control active parallel workers using `--concurrency <N>` (default 10 workers).
  * **Per-Request Timeout:** FuzzSpec enforces a strict 10-second per-request timeout to prevent hanging connections.

### 3. Downstream Third-Party Side Effects & Billing Surges
* **Risk:** Fuzzing endpoints that trigger external transactional webhooks (e.g., email verification via SendGrid, SMS OTP via Twilio, payment gateway charges via Stripe) can send spam or incur unexpected billing costs.
* **Mitigation:**
  * Mock or stub third-party drivers in your test environment (e.g., use local Mailpit/Mailhog for emails, mock adapters for payment gateways).

### 4. Server-Side Request Forgery (SSRF) in MCP
* **Risk:** If an unconstrained AI agent passes malicious internal addresses (e.g., Cloud Instance Metadata `http://169.254.169.254/` or internal database sockets) to the MCP server.
* **Mitigation:**
  * FuzzSpec validates URL schemes (`http://` and `https://`) and rejects malformed protocol bindings.

---

## 📋 Pre-Flight Security Checklist

Before running FuzzSpec in CI/CD pipelines or local development environments, verify this checklist:

| Verification Item | Safe Standard | Operational Note |
| :--- | :---: | :--- |
| **Target URL** | ✅ Local / Staging | Verify the target is not a production URL |
| **Database State** | ✅ Seeded / Ephemeral | Ensure the database can be reset safely |
| **Third-Party Integrations** | ✅ Mocked / Disabled | Stub payment gateways, SMS, and email services |
| **Auth Credentials** | ✅ Injected via `token_env` | Use test-scoped credentials with minimal privileges |
| **Safe Mode Initial Run** | ✅ `--safe-mode=true` | Perform initial smoke fuzzing with read-only methods |
| **cURL Reproducers** | ✅ Replace `[REDACTED]` | Manually insert test token when reproducing bugs locally |

---

## 🐛 Reporting a Vulnerability

If you discover a security vulnerability in FuzzSpec (such as a sanitizer bypass or context leak):

1. **Do NOT open a public GitHub issue.**
2. Send a report via **GitHub Private Vulnerability Reporting** on the repository: [https://github.com/hanifalkauni/fuzzspec/security/advisories/new](https://github.com/hanifalkauni/fuzzspec/security/advisories/new)
3. Or email the maintainer directly at: `m.hanif.alkauni@gmail.com`

Please include:
* Description of the vulnerability and attack vector.
* Minimal reproduction steps or proof-of-concept (PoC).
* Impact assessment.

We commit to acknowledging your report within **48 hours** and providing regular updates until a security patch is released.
