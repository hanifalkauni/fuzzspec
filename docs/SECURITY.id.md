# Kebijakan Keamanan & Model Ancaman (Security Policy & Threat Model)

Di **FuzzSpec**, keamanan adalah prinsip dasar. Karena FuzzSpec berinteraksi dengan spesifikasi OpenAPI, mengeksekusi pengujian HTTP fuzzing secara live, dan terhubung dengan AI coding agent (Cursor, Claude, Copilot, Antigravity, Windsurf), kami menerapkan arsitektur **Zero-Trust Security & Defense-in-Depth** untuk melindungi kredensial, infrastruktur, dan layanan backend Anda.

---

## 🛡️ Versi yang Didukung

Kami secara aktif menyediakan patch keamanan dan perbaikan bug untuk versi berikut:

| Versi | Didukung | Keterangan |
| :--- | :---: | :--- |
| `v1.x.x` | ✅ Ya | Rilis resmi saat ini |
| `< 1.0.0` | ❌ Tidak | Build prarilis/beta |

---

## 🔒 5 Lapisan Perlindungan Kredensial Zero-Trust

FuzzSpec mengimplementasikan 5 lapisan keamanan untuk memastikan API token, URI database, private key, dan environment variable **tidak pernah bocor** ke log percakapan AI, jendela konteks prompt, komentar PR CI/CD, maupun repositori publik:

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                    5 LAPISAN PERLINDUNGAN KREDENSIAL FUZZSPEC                   │
├─────────────────────────────────────────────────────────────────────────────────┤
│ 1. Agent Boundary   : .cursorignore & .claudeignore mengisolasi .env & rahasia  │
│ 2. Scanner Blacklist: Parser mengabaikan .git, .aws, .ssh, *.pem, credentials  │
│ 3. RAM-Only Auth    : token_env membaca token dinamis hanya ke dalam RAM OS     │
│ 4. Stream Redactor  : Otomatis menyensor Bearer token, API key, URI ke REDACTED │
│ 5. Prompt Scrubbing : LLM HANYA menerima tipe skema, TANPA auth/token mentah    │
└─────────────────────────────────────────────────────────────────────────────────┘
```

1. **Isolasi Konteks Agent (`.cursorignore`, `.claudeignore`, `.gitignore`):**  
   File seperti `.env*`, `credentials.json`, `*.pem`, `*.key`, `id_rsa`, dan direktori `.aws` otomatis diblokir dari proses indexing, search embeddings, dan context window AI IDE.
2. **Blacklist Scanner Engine (`tools_scan.go`, `discover.go`):**  
   Mesin auto-discovery OpenAPI dan route discovery secara aktif melompati file dan direktori sensitif.
3. **Injeksi Token via RAM-Only (`token_env`):**  
   FuzzSpec melarang penulisan token rahasia secara mentah di file konfigurasi (`fuzzspec.yaml`). Token direferensikan via nama variabel lingkungan (contoh: `token_env: "AUTH_TOKEN"`) dan dibaca langsung dari memori OS saat request HTTP ditembakkan.
4. **Real-Time Stream Redactor (`sanitizer.go`):**  
   Semua output terminal, respons tool MCP, ekspor SARIF v2.1.0, laporan JUnit XML, komentar GitHub PR, dan cURL reproducer diproses melalui multi-regex redactor yang menyensor Bearer token, OpenAI/Anthropic/AWS/GitHub API key, connection string database, dan password menjadi `[REDACTED]`.
5. **Isolasi Prompt LLM (`ai_generator.go`):**  
   Ketika membuat test case boundary dan adversarial dengan LLM (Gemini, OpenAI, Claude), FuzzSpec **hanya mengirimkan struktur skema OpenAPI** (nama field, tipe data, enum, batasan nilai). Tidak ada authorization header, session cookie, atau data database produksi yang dikirim ke penyedia AI.

---

## ⚠️ Model Ancaman & Resiko Operasional

Ketika menjalankan pengujian kontrak API dan adversarial fuzzing otomatis terhadap web service yang sedang berjalan, perhatikan resiko operasional berikut beserta mitigasinya:

### 1. Resiko Kerusakan Data / State Corruption
* **Resiko:** Fuzzing otomatis pada metode HTTP non-idempotent (`POST`, `PUT`, `PATCH`, `DELETE`) dengan boundary ID atau payload adversarial berpotensi mengubah atau menghapus data riil.
* **Mitigasi:**
  * **Safe Mode:** Gunakan `--safe-mode=true` untuk membatasi fuzzing hanya pada metode read-only (`GET`, `HEAD`, `OPTIONS`).
  * **Lingkungan Terisolasi:** Selalu jalankan FuzzSpec pada container lokal, server staging, atau database uji coba terisolasi. **Jangan pernah menjalankan destructive fuzzing langsung pada database produksi.**

### 2. Denial of Service (DoS) & Kehabisan Sumber Daya
* **Resiko:** Konkurensi tinggi, laju request yang terlalu cepat, atau payload ekstrem (seperti INT64 overflow, JSON bertingkat sangat dalam, atau string ReDoS) dapat mengunci CPU, menghabiskan memory (OOM), atau menghabiskan connection pool database.
* **Mitigasi:**
  * **Rate Limiting:** Batasi laju request dengan `--rps <N>` (algoritma token bucket, default: 50 RPS).
  * **Bounded Concurrency:** Batasi jumlah worker paralel aktif dengan `--concurrency <N>` (default: 10 worker).
  * **Timeout per Request:** FuzzSpec menerapkan batas timeout ketat 10 detik per request agar koneksi tidak menggantung (*hang*).

### 3. Efek Samping Layanan Pihak Ketiga & Lonjakan Biaya
* **Resiko:** Menguji endpoint yang memicu webhook eksternal (misal: verifikasi email via SendGrid, SMS OTP via Twilio, atau tagihan payment gateway via Stripe) dapat mengirim pesan sampah atau memicu lonjakan tagihan tak terduga.
* **Mitigasi:**
  * Gunakan Mocking / Stubbing untuk driver pihak ketiga di lingkungan pengujian (misal: gunakan Mailpit/Mailhog untuk email, adapter mock untuk payment gateway).

### 4. Server-Side Request Forgery (SSRF) pada MCP
* **Resiko:** Jika AI agent memberikan alamat internal berbahaya (misal: Cloud Metadata IP `http://169.254.169.254/` atau socket database internal) ke MCP server.
* **Mitigasi:**
  * FuzzSpec memvalidasi skema protokol URL (`http://` dan `https://`) serta menolak format binding protokol yang tidak valid.

---

## 📋 Checklist Keamanan Sebelum Menjalankan FuzzSpec

Sebelum menjalankan FuzzSpec di CI/CD pipeline atau environment lokal Anda, pastikan checklist berikut terpenuhi:

| Item Verifikasi | Standar Aman | Catatan Operasional |
| :--- | :---: | :--- |
| **Target URL** | ✅ Lokal / Staging | Pastikan target bukan domain produksi live |
| **Status Database** | ✅ Seeded / Ephemeral | Pastikan database dapat di-reset dengan aman |
| **Layanan Pihak Ketiga** | ✅ Dimock / Dimatikan | Nonaktifkan payment gateway, SMS, dan email live |
| **Kredensial Autentikasi** | ✅ Diinjeksi via `token_env` | Gunakan token khusus testing dengan hak akses minimal |
| **Uji Coba Awal** | ✅ `--safe-mode=true` | Jalankan smoke test awal dengan metode read-only |
| **cURL Reproducer** | ✅ Ganti `[REDACTED]` | Masukkan token testing Anda secara manual saat mereproduksi bug di terminal |

---

## 🐛 Pelaporan Kerentanan Keamanan

Jika Anda menemukan celah keamanan (*vulnerability*) pada FuzzSpec (seperti bypass sanitizer atau kebocoran konteks):

1. **JANGAN membuat issue publik di GitHub.**
2. Laporkan secara privat melalui **GitHub Private Vulnerability Reporting** di: [https://github.com/hanifalkauni/fuzzspec/security/advisories/new](https://github.com/hanifalkauni/fuzzspec/security/advisories/new)
3. Atau kirim email langsung ke maintainer di: `m.hanif.alkauni@gmail.com`

Mohon sertakan:
* Deskripsi kerentanan dan vektor serangan.
* Langkah reproduksi minimal atau Proof-of-Concept (PoC).
* Analisis dampak potensi risiko.

Kami berkomitmen untuk merespons laporan Anda dalam waktu **48 jam** dan memberikan pembaruan berkala hingga patch keamanan dirilis.
