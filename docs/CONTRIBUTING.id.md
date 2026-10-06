# Panduan Kontribusi FuzzSpec (Contributing Guide)

Terima kasih atas minat Anda untuk berkontribusi pada **FuzzSpec**! Kami sangat menyambut kontribusi dari para pengembang, QA engineer, dan peneliti keamanan di seluruh komunitas open-source.

---

## 🛠️ Prasyarat Pengembangan

* **Go**: versi `1.24` atau lebih baru ([https://go.dev/dl/](https://go.dev/dl/))
* **Node.js**: versi `18.0.0` atau lebih baru (untuk runner NPX dan pengujian adapter)
* **Git**: versi `2.30+`

---

## 🚀 Memulai Pengembangan

### 1. Clone Repositori
```bash
git clone https://github.com/hanifalkauni/fuzzspec.git
cd fuzzspec
```

### 2. Menjalankan Seluruh Test Suite
Pastikan seluruh unit test, oracles, dan detektor race condition konkurensi berjalan bersih:
```bash
go test -race -v ./...
```

### 3. Build Binary Lokal
```bash
# Kompilasi CLI dan MCP binary
go build -o fuzzspec.exe ./cmd/fuzzspec
go build -o fuzzspec-mcp.exe ./cmd/fuzzspec-mcp

# Atau menggunakan Makefile
make build
```

---

## 🧩 Area Kontribusi

### 1. Menambahkan Aturan Mutasi Heuristik Baru
Aturan mutasi heuristik berada di [`internal/generator/rule_mutator.go`](../internal/generator/rule_mutator.go).
* Boundary underflow/overflow numerik.
* Vektor injeksi polyglot (SQLi, NoSQL, CRLF, Path Traversal, Null byte).
* Mutasi format khusus (tanggal, UUID, email, regex kustom).

### 2. Menambahkan Detektor Stack Trace / Panic Framework Baru
Aturan deteksi stack trace runtime berada di [`internal/oracle/polyglot_scanner.go`](../internal/oracle/polyglot_scanner.go).
* Tambahkan tanda tangan regex untuk menangkap runtime panic di bahasa atau framework baru (misal: Elixir Phoenix, Kotlin Ktor, Swift Vapor).

### 3. Mengembangkan Exporter & Integrasi
Format pelaporan enterprise berada di [`internal/reporter/`](../internal/reporter/):
* `sarif.go` (SARIF v2.1.0)
* `junit.go` (JUnit XML)
* `markdown.go` (Ringkasan PR GitHub)
* `json.go` (Log diagnostik)

---

## 📋 Pedoman Pull Request & Commit

1. **Buat Branch Fitur Baru:**
   ```bash
   git checkout -b feat/nama-fitur-baru
   ```
2. **Format Pesan Commit (Conventional Commits):**
   * `feat: add Elixir Phoenix panic detector to polyglot oracle`
   * `fix: handle circular $ref dereferencing in nullable oneOf schemas`
   * `docs: update environment variables table in README`
   * `test: add boundary test case for int64 overflows`
3. **Pastikan Kualitas Kode:**
   * Jalankan `go test -race -v ./...`
   * Format kode dengan `gofmt -s -w .`
4. **Buka Pull Request:**
   * Berikan deskripsi perubahan yang jelas, motivasi, dan bukti hasil tes.

---

## 🔒 Pelaporan Kerentanan Keamanan

Jika Anda menemukan celah keamanan (seperti bypass sanitizer atau kebocoran kredensial), mohon **jangan membuka issue publik di GitHub**. Ikuti [Kebijakan Keamanan (SECURITY.md)](../SECURITY.md) dan laporkan via:
* **GitHub Private Vulnerability Reporting**: [https://github.com/hanifalkauni/fuzzspec/security/advisories/new](https://github.com/hanifalkauni/fuzzspec/security/advisories/new)
* **Email Maintainer**: `m.hanif.alkauni@gmail.com`
