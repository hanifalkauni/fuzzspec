# Contributing to FuzzSpec

Thank you for your interest in contributing to **FuzzSpec**! We welcome contributions from developers, QA engineers, and security researchers across the open-source community.

---

## 🛠️ Development Prerequisites

* **Go**: `1.24` or higher ([https://go.dev/dl/](https://go.dev/dl/))
* **Node.js**: `18.0.0` or higher (for NPX runner and adapter tests)
* **Git**: `2.30+`

---

## 🚀 Getting Started

### 1. Clone the Repository
```bash
git clone https://github.com/hanifalkauni/fuzzspec.git
cd fuzzspec
```

### 2. Verify and Run the Test Suite
Ensure all unit tests, oracles, and concurrency race detectors pass cleanly:
```bash
go test -race -v ./...
```

### 3. Build Local Binaries
```bash
# Build CLI and MCP binaries
go build -o fuzzspec.exe ./cmd/fuzzspec
go build -o fuzzspec-mcp.exe ./cmd/fuzzspec-mcp

# Or using Makefile
make build
```

---

## 🧩 Areas to Contribute

### 1. Adding New Heuristic Mutation Rules
Heuristic mutation rules are located in [`internal/generator/rule_mutator.go`](./internal/generator/rule_mutator.go).
* Boundary underflows/overflows.
* Polyglot injection vectors (SQLi, NoSQL, CRLF, Path Traversal, Null byte).
* Format-specific boundary inputs (dates, UUIDs, emails, custom regex patterns).

### 2. Adding Native Polyglot Stack Trace / Panic Scanners
Runtime stack trace detection rules are defined in [`internal/oracle/polyglot_scanner.go`](./internal/oracle/polyglot_scanner.go).
* Add regex signatures for unhandled panics and exceptions in new languages or frameworks (e.g., Elixir Phoenix, Kotlin Ktor, Swift Vapor).

### 3. Extending Exporters & Integrations
Enterprise reporters reside in [`internal/reporter/`](./internal/reporter/):
* `sarif.go` (SARIF v2.1.0)
* `junit.go` (JUnit XML)
* `markdown.go` (PR summary)
* `json.go` (Diagnostic logs)

---

## 📋 Pull Request & Commit Guidelines

1. **Create a Feature Branch:**
   ```bash
   git checkout -b feat/my-new-feature
   ```
2. **Commit Message Format (Conventional Commits):**
   * `feat: add Elixir Phoenix panic detector to polyglot oracle`
   * `fix: handle circular $ref dereferencing in nullable oneOf schemas`
   * `docs: update environment variables table in README`
   * `test: add boundary test case for int64 overflows`
3. **Ensure Code Quality:**
   * Run `go test -race -v ./...`
   * Format code with `gofmt -s -w .`
4. **Open a Pull Request:**
   * Provide a clear description of changes, motivation, and test coverage evidence.

---

## 🔒 Reporting Security Vulnerabilities

If you discover a security vulnerability (such as a sanitizer bypass or credential leakage), please **do not open a public GitHub issue**. Follow our [Security Policy (SECURITY.md)](./SECURITY.md) and report via:
* **GitHub Private Vulnerability Reporting**: [https://github.com/hanifalkauni/fuzzspec/security/advisories/new](https://github.com/hanifalkauni/fuzzspec/security/advisories/new)
* **Maintainer Email**: `m.hanif.alkauni@gmail.com`
