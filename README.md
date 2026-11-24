# 🏪 Fainaya Store Support

> **Open-Source Retail IT Operations & Helpdesk Platform**  
> Solusi *Self-Hosted / Local-First* untuk operasional TI toko ritel: manajemen tiket POS/Printer, checklist harian (*Daily IT Readiness*), pemantauan Windows Server, status antivirus, inventaris perangkat keras, diagnostik jaringan LAN/Wi-Fi, dan eskalasi ke Tim Kantor Pusat (HO) / Vendor eksternal.

Dilengkapi dengan **Adaptive 6-Theme Design Engine** berbasis *CSS Custom Properties* (*Design Tokens*) yang mendukung pergantian tema seketika (*zero page reload*).

---

## 📑 Daftar Isi

- [Fitur Utama](#-fitur-utama)
- [Arsitektur & Tumpukan Teknologi](#-arsitektur--tumpukan-teknologi)
- [Adaptive 6-Theme Engine](#-adaptive-6-theme-engine)
- [Keamanan & RBAC (Role-Based Access Control)](#-keamanan--rbac-role-based-access-control)
- [Prasyarat Sistem](#-prasyarat-sistem)
- [Panduan Instalasi & Menjalankan Lokal](#-panduan-instalasi--menjalankan-lokal)
- [Akun Bawaan Demo](#-akun-bawaan-demo)
- [Struktur Proyek](#-struktur-proyek)
- [Lisensi](#-lisensi)

---

## ✨ Fitur Utama

1. **Modul Dukungan TI Harian (Helpdesk POS & Printer):**
   - Pelaporan kendala satu klik untuk kasir (*Quick Ticket*).
   - Pengelompokan urgensi (*Critical, High, Medium, Low*).
2. **Checklist Kesiapan TI Harian (*Daily IT Readiness*):**
   - Daftar periksa otomatis *Opening* dan *Closing* toko (koneksi EDC, printer struk, UPS, dsb).
3. **Manajemen Hardware & Aset:**
   - Registri lengkap PC POS, Thermal Printer, Scanner, UPS, nomor seri, masa garansi, dan jadwal pemeliharaan berkala.
4. **Software & Antivirus Health Tracker:**
   - Katalog installer lokal tanpa membebani bandwidth internet toko.
   - Pemantauan status definisi antivirus, tanggal scan, dan status kesehatan terminal kasir.
5. **Diagnostik Jaringan (LAN & Wi-Fi):**
   - Heartbeat monitoring IP Gateway, Switch, AP, EDC, dan Printer.
   - Utilitas diagnostik web bawaan (*Ping, DNS Lookup, TCP Port Check*) yang aman dari ancaman *Command Injection*.
6. **Pemantauan & Kontrol Aman Windows Server:**
   - Metrik CPU, RAM, kapasitas partisi disk (C: & D:), uptime sistem, dan verifikasi backup database POS.
   - Kontrol layanan Windows (*Start/Restart Service*) khusus daftar aman (*whitelisted services*).
7. **Basis Pengetahuan & SOP Mandiri:**
   - Panduan praktis penanganan kendala umum (kertas struk macet, restart aman aplikasi kasir).
8. **Eskalasi ke Kantor Pusat (HO) & Vendor SLA:**
   - Direktori kontak darurat PIC vendor dan eskalasi satu klik berlampirkan log diagnostik.
9. **Jejak Audit Sistem (*Immutable Audit Trail*):**
   - Pencatatan seluruh aksi teknis, login, dan kontrol service lengkap dengan timestamp dan IP intranet.

---

## 🛠️ Arsitektur & Tumpukan Teknologi

Mengikuti panduan dokumentasi resmi untuk arsitektur *Local-First* berperforma tinggi:

- **Backend:** [Go](https://go.dev/) dengan [Echo Framework v4](https://echo.labstack.com/)
  - Kompilasi menjadi *single-binary executable* (`fainaya-server.exe`) yang ringan (< 80 MB RAM).
  - Melayani REST API sekaligus aset statis frontend SPA.
- **Database:** [SQLite](https://sqlite.org/) (driver murni Go `modernc.org/sqlite`)
  - Berjalan dalam mode **WAL (Write-Ahead Logging)** untuk konkurensi tinggi tanpa server DB terpisah.
- **Frontend:** [Svelte](https://svelte.dev/) + [Vite](https://vite.dev/) (TypeScript)
  - Desain reaktif tanpa overhead *virtual DOM*, ukuran bundle sangat ringkas.
- **Styling:** CSS Custom Properties (*Design Tokens*) untuk sistem 6 tema adaptif.

---

## 🎨 Adaptive 6-Theme Engine

Aplikasi menyediakan 6 varian tema visual yang berganti seketika (*zero page reload* $< 16\text{ ms}$) melalui manipulasi atribut `[data-theme]` pada elemen `<html>`:

1. **Daylight Crisp (Default Light):** Kontras tinggi untuk area kasir & lantai toko yang terang benderang.
2. **Midnight NOC (Dark Mode):** Meredam silau untuk pemantauan malam hari atau ruang server (*back office*).
3. **High-Contrast POS (Accessibility):** Tampilan hitam-kuning pekat untuk monitor POS tabung / layar pantulan cahaya.
4. **Nordic Slate (Cool Calm):** Nuansa sejuk abu-abu lembut untuk input audit inventaris dalam durasi panjang.
5. **Warm Parchment (Eye-Comfort):** Warna krem kertas lembut untuk membaca dokumen SOP/Knowledge Base.
6. **Terminal Ops (Classic Tech):** Monospace hijau fosfor untuk pemantauan log server dan diagnostik teknis.

---

## 🔒 Keamanan & RBAC (Role-Based Access Control)

Sesuai standar praktik keamanan dokumentasi Go, Echo, OWASP, dan JWT:

1. **Perlindungan Data Sensitif:**
   - Seluruh kata sandi di-hash menggunakan algoritma **bcrypt** dengan cost yang aman.
   - Database lokal (`*.db`), berkas WAL (`*.db-wal`), dan berkas environment `.env` secara eksplisit dikecualikan dari Git melalui `.gitignore`.
2. **Autentikasi & Sesi:**
   - Autentikasi berbasis **JSON Web Token (JWT)** bertanda tangan HS256 dengan masa kedaluwarsa 24 jam.
   - Sesi didukung oleh header `Authorization: Bearer <token>` dan *HttpOnly Cookie* (`fainaya_token`) dengan atribut `SameSite=Lax`.
3. **Pemisahan Hak Akses (RBAC):**
   - `store_user`: Melaporkan tiket & melihat tiket miliknya sendiri.
   - `it_support`: CRUD tiket, checklist harian, diagnostik jaringan, kontrol service, inventaris aset.
   - `store_manager`: Memantau operasional, persetujuan eskalasi, dan meninjau laporan.
   - `ho_it`: Menerima tiket eskalasi, pemantauan server menyeluruh.
   - `vendor`: Hanya mengakses tiket yang dieskalasikan/ditugaskan ke vendor.
   - `super_admin`: Akses penuh ke seluruh konfigurasi sistem dan manajemen user.
4. **Pencegahan Command Injection:**
   - Utilitas diagnostik jaringan membersihkan input dari karakter kontrol shell (`&`, `|`, `;`, `` ` ``, `$`, `>`, `<`).
   - Eksekusi layanan Windows dibatasi secara ketat hanya pada *whitelisted services*.

---

## 📦 Prasyarat Sistem

- **OS:** Windows 10/11 / Windows Server 2016+, atau Linux/macOS.
- **Go:** Versi 1.25+ (untuk mengkompilasi backend).
- **Node.js:** Versi 18+ & npm (untuk mengkompilasi frontend Svelte).

---

## 🚀 Panduan Instalasi & Menjalankan Lokal

### 1. Klon Repositori
```bash
git clone https://github.com/your-username/fainaya-store-support.git
cd fainaya-store-support
```

### 2. Kompilasi Frontend Svelte
```bash
cd client
npm install
npm run build
cd ..
```
*Hasil kompilasi akan otomatis dibuat di folder `client/dist`.*

### 3. Kompilasi & Jalankan Backend Go
```bash
cd server
go build -o fainaya-server.exe .
.\fainaya-server.exe
```

Aplikasi web sekarang aktif dan dapat diakses di browser:
👉 **`http://localhost:8080`** (atau via IP intranet `http://192.168.x.x:8080`)

---

## 👥 Akun Bawaan Demo

Basis data secara otomatis melakukan *initial seeding* saat pertama kali dijalankan:

| Username | Password | Peran (*Role*) | Deskripsi |
| :--- | :--- | :--- | :--- |
| `admin` | `admin123` | `super_admin` | Fainaya Super Administrator |
| `it_support` | `support123` | `it_support` | Teknisi TI Toko (*Store IT*) |
| `kasir1` | `kasir123` | `store_user` | Dewi Sartika (Kasir 1) |
| `store_mgr` | `mgr123` | `store_manager` | Kepala Toko (*Store Manager*) |
| `ho_it` | `ho123` | `ho_it` | Tim TI Kantor Pusat (*Tier-2*) |
| `vendor_edc`| `vendor123` | `vendor` | Teknisi Mitra EDC Merchant |

---

## 📂 Struktur Proyek

```text
fainaya-sp/
├── client/                     # Frontend Svelte + Vite (TypeScript)
│   ├── src/
│   │   ├── api.ts              # Fetch helper & token bearer management
│   │   ├── app.css             # Adaptive 6-Theme Design Engine (CSS Tokens)
│   │   ├── App.svelte          # Komponen Utama Dashboard & Modul PRD
│   │   ├── main.ts             # Entrypoint Svelte
│   │   └── types.ts            # Definisi TypeScript Interface
│   ├── package.json
│   └── vite.config.ts
├── server/                     # Backend Go (Echo Framework)
│   ├── auth.go                 # Middleware JWT & RBAC Role Validation
│   ├── db.go                   # Skema SQLite WAL & Initial Data Seeder
│   ├── handlers.go             # Handler API Modul 1 - 7 PRD
│   ├── main.go                 # Server Setup, Routing & Static File Serving
│   └── go.mod
├── .gitignore                  # Pengecualian berkas sensitif, binary & database
├── fainaya_store_support_prd.md# Dokumen Kebutuhan Produk (PRD)
└── README.md                   # Dokumentasi instalasi, arsitektur & keamanan
```

---

## 📄 Lisensi

Platform ini dirilis di bawah lisensi **MIT / Apache 2.0 Open Source License**. Bebas dimodifikasi dan didistribusikan untuk kebutuhan toko ritel.

---
<img width="1412" height="908" alt="image" src="https://github.com/user-attachments/assets/fbfe40cf-c06e-4cc4-aab1-9fcd12d5c08d" />

---
<img width="1897" height="900" alt="image" src="https://github.com/user-attachments/assets/d111320f-90e5-4da4-bbf7-96aa610ffb0b" />

---
<img width="1892" height="897" alt="image" src="https://github.com/user-attachments/assets/af537473-6189-4981-885a-0fc56a9a8ec1" />

---
<img width="1897" height="896" alt="image" src="https://github.com/user-attachments/assets/12dd4421-2724-427c-a668-558de5b1c870" />

---
<img width="1901" height="902" alt="image" src="https://github.com/user-attachments/assets/1ab3b982-9a1a-4200-9078-c1c7148632dc" />

---
<img width="1887" height="898" alt="image" src="https://github.com/user-attachments/assets/4fe605b8-39c3-4c06-aa20-37bc116d4bce" />

---
<img width="1892" height="900" alt="image" src="https://github.com/user-attachments/assets/f8909647-287e-42cf-9f53-2f8a3032ee75" />



