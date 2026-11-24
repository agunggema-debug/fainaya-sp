package main

import (
	"database/sql"
	"log"
	"os"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

func initDB() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./fainaya.db"
	}

	var err error
	// Enable WAL mode for high concurrency in SQLite
	db, err = sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	createTables()
	seedInitialData()
}

func createTables() {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		full_name TEXT NOT NULL,
		role TEXT NOT NULL,
		store_code TEXT NOT NULL DEFAULT 'STORE-001',
		theme_pref TEXT NOT NULL DEFAULT 'daylight-crisp',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS tickets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ticket_no TEXT UNIQUE NOT NULL,
		category TEXT NOT NULL,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		urgency TEXT NOT NULL DEFAULT 'Medium',
		status TEXT NOT NULL DEFAULT 'Open',
		device_name TEXT,
		reported_by_id INTEGER NOT NULL,
		assigned_to_id INTEGER,
		escalated_to TEXT,
		escalation_notes TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (reported_by_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS checklists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		shift_type TEXT NOT NULL, -- 'Opening' or 'Closing'
		check_date TEXT NOT NULL,
		item_name TEXT NOT NULL,
		category TEXT NOT NULL,
		is_checked INTEGER NOT NULL DEFAULT 0,
		notes TEXT,
		checked_by_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (checked_by_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS assets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		asset_code TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		category TEXT NOT NULL, -- PC, POS, Thermal Printer, Barcode Scanner, UPS, etc.
		serial_number TEXT,
		location TEXT NOT NULL, -- Kasir 1, Back Office, Gudang
		status TEXT NOT NULL DEFAULT 'Good', -- Good, Maintenance, Defective
		specs TEXT,
		warranty_until TEXT,
		last_maintenance TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS software_catalog (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		version TEXT NOT NULL,
		category TEXT NOT NULL, -- POS, Driver, Antivirus, Utility
		file_path TEXT,
		download_url TEXT,
		status TEXT NOT NULL DEFAULT 'Available',
		description TEXT
	);

	CREATE TABLE IF NOT EXISTS antivirus_status (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hostname TEXT UNIQUE NOT NULL,
		ip_address TEXT NOT NULL,
		antivirus_name TEXT NOT NULL,
		service_active INTEGER NOT NULL DEFAULT 1,
		defs_updated_at TEXT NOT NULL,
		last_scan_at TEXT NOT NULL,
		threats_found INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'Healthy' -- Healthy, Warning, Critical
	);

	CREATE TABLE IF NOT EXISTS network_devices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		ip_address TEXT UNIQUE NOT NULL,
		device_type TEXT NOT NULL, -- Gateway, Switch, AP, EDC, Printer
		vlan TEXT,
		mac_address TEXT,
		status TEXT NOT NULL DEFAULT 'Online', -- Online, Offline
		last_ping_ms REAL DEFAULT 0,
		last_checked DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS server_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hostname TEXT NOT NULL,
		cpu_usage REAL NOT NULL,
		ram_usage REAL NOT NULL,
		disk_c_free_pct REAL NOT NULL,
		uptime_hours REAL NOT NULL,
		recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS windows_services (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		service_name TEXT UNIQUE NOT NULL,
		display_name TEXT NOT NULL,
		status TEXT NOT NULL, -- Running, Stopped
		is_whitelisted INTEGER NOT NULL DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS activity_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		username TEXT NOT NULL,
		activity_type TEXT NOT NULL,
		description TEXT NOT NULL,
		ip_address TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS knowledge_base (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		category TEXT NOT NULL,
		summary TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS vendor_contacts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor_name TEXT NOT NULL,
		service_type TEXT NOT NULL,
		contact_person TEXT,
		phone TEXT NOT NULL,
		email TEXT,
		sla_response TEXT,
		contract_info TEXT
	);
	`

	_, err := db.Exec(schema)
	if err != nil {
		log.Fatalf("Failed to execute database migration: %v", err)
	}
}

func seedInitialData() {
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if count > 0 {
		return
	}

	log.Println("Seeding initial data...")

	// Helper to hash passwords using bcrypt
	hashPassword := func(pw string) string {
		h, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
		return string(h)
	}

	// 1. Users with predefined roles
	users := []struct {
		username, password, fullName, role, storeCode string
	}{
		{"admin", "admin123", "Fainaya Super Administrator", "super_admin", "STORE-001"},
		{"kasir1", "kasir123", "Dewi Sartika (Kasir 1)", "store_user", "STORE-001"},
		{"it_support", "support123", "Rian Hidayat (Store IT)", "it_support", "STORE-001"},
		{"store_mgr", "mgr123", "Budi Santoso (Store Manager)", "store_manager", "STORE-001"},
		{"ho_it", "ho123", "Ari Wibowo (HO IT Tier-2)", "ho_it", "HO-CENTRAL"},
		{"vendor_edc", "vendor123", "Teknisi Vendor EDC", "vendor", "STORE-001"},
	}

	stmt, err := db.Prepare("INSERT INTO users (username, password_hash, full_name, role, store_code) VALUES (?, ?, ?, ?, ?)")
	if err == nil {
		for _, u := range users {
			_, _ = stmt.Exec(u.username, hashPassword(u.password), u.fullName, u.role, u.storeCode)
		}
		stmt.Close()
	}

	// 2. Initial Whitelisted Windows Services
	services := []struct {
		name, display string
	}{
		{"MSSQL$POSDB", "SQL Server (POS Database)"},
		{"Spooler", "Print Spooler"},
		{"LanmanServer", "Server File Sharing (SMB)"},
		{"Dhcp", "DHCP Client / Service"},
		{"wuauserv", "Windows Update"},
	}
	sStmt, _ := db.Prepare("INSERT INTO windows_services (service_name, display_name, status, is_whitelisted) VALUES (?, ?, 'Running', 1)")
	if sStmt != nil {
		for _, s := range services {
			_, _ = sStmt.Exec(s.name, s.display)
		}
		sStmt.Close()
	}

	// 3. Initial Network Devices
	netDevices := []struct {
		name, ip, dtype, vlan, mac string
	}{
		{"Main Gateway Router", "192.168.10.1", "Gateway", "VLAN 10", "00:1A:2B:3C:4D:01"},
		{"Switch Core 24-Port", "192.168.10.2", "Switch", "VLAN 10", "00:1A:2B:3C:4D:02"},
		{"Access Point Kasir", "192.168.10.5", "AP", "VLAN 20", "00:1A:2B:3C:4D:05"},
		{"EDC Merchant Terminal 1", "192.168.10.50", "EDC", "VLAN 20", "00:1A:2B:3C:4D:50"},
		{"Thermal Printer Kasir 1", "192.168.10.101", "Printer", "VLAN 20", "00:1A:2B:3C:4D:65"},
		{"Server POS Lokal", "192.168.10.10", "Server", "VLAN 10", "00:1A:2B:3C:4D:10"},
	}
	nStmt, _ := db.Prepare("INSERT INTO network_devices (name, ip_address, device_type, vlan, mac_address, status, last_ping_ms) VALUES (?, ?, ?, ?, ?, 'Online', 1.2)")
	if nStmt != nil {
		for _, d := range netDevices {
			_, _ = nStmt.Exec(d.name, d.ip, d.dtype, d.vlan, d.mac)
		}
		nStmt.Close()
	}

	// 4. Initial Antivirus Status
	avs := []struct {
		hostname, ip, avName, defs, scan, status string
		threats, active                           int
	}{
		{"POS-KASIR-01", "192.168.10.101", "Windows Defender Enterprise", "2026-11-23", "2026-11-24 06:00", "Healthy", 0, 1},
		{"POS-KASIR-02", "192.168.10.102", "Windows Defender Enterprise", "2026-11-23", "2026-11-24 06:00", "Healthy", 0, 1},
		{"BACKOFFICE-PC", "192.168.10.20", "Windows Defender Enterprise", "2026-11-20", "2026-11-21 12:00", "Warning", 0, 1},
	}
	avStmt, _ := db.Prepare("INSERT INTO antivirus_status (hostname, ip_address, antivirus_name, service_active, defs_updated_at, last_scan_at, threats_found, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
	if avStmt != nil {
		for _, a := range avs {
			_, _ = avStmt.Exec(a.hostname, a.ip, a.avName, a.active, a.defs, a.scan, a.threats, a.status)
		}
		avStmt.Close()
	}

	// 5. Initial Assets
	assets := []struct {
		code, name, cat, sn, loc, status, specs, war string
	}{
		{"AST-POS-01", "Terminal POS Kasir 1", "POS", "SN-POS-88219", "Kasir 1", "Good", "Core i3 / 8GB RAM / 256GB SSD", "2027-12-31"},
		{"AST-POS-02", "Terminal POS Kasir 2", "POS", "SN-POS-88220", "Kasir 2", "Good", "Core i3 / 8GB RAM / 256GB SSD", "2027-12-31"},
		{"AST-PRN-01", "Epson TM-T82X Thermal Printer", "Thermal Printer", "SN-EP-33411", "Kasir 1", "Good", "Thermal 80mm Auto-cutter USB/LAN", "2026-12-31"},
		{"AST-UPS-01", "APC Back-UPS 650VA", "UPS", "SN-APC-0091", "Kasir 1", "Good", "390 Watts / 650 VA", "2027-06-30"},
	}
	astStmt, _ := db.Prepare("INSERT INTO assets (asset_code, name, category, serial_number, location, status, specs, warranty_until, last_maintenance) VALUES (?, ?, ?, ?, ?, ?, ?, ?, '2026-11-15')")
	if astStmt != nil {
		for _, ast := range assets {
			_, _ = astStmt.Exec(ast.code, ast.name, ast.cat, ast.sn, ast.loc, ast.status, ast.specs, ast.war)
		}
		astStmt.Close()
	}

	// 6. Initial Knowledge Base
	kbs := []struct {
		title, cat, sum, content string
	}{
		{
			"Mengatasi Kertas Macet (Paper Jam) Printer Struk Thermal",
			"Printer",
			"Langkah cepat dan aman mengeluarkan kertas tersangkut tanpa merusak head printer.",
			"1. Matikan saklar power printer.\n2. Buka tuas penutup printer perlahan.\n3. Jika macet kuat, putar pemotong manual di bawah penutup kecil.\n4. Tarik kertas perlahan ke arah keluar roll.\n5. Pasang kembali gulungan kertas dengan posisi permukaan menghadap ke atas.\n6. Nyalakan kembali printer dan lakukan Self-Test (tahan Feed lalu nyalakan).",
		},
		{
			"Restart Aman Aplikasi Kasir & Database POS Lokal",
			"Aplikasi POS",
			"Prosedur penutupan paksa dan reset session bila aplikasi kasir macet.",
			"1. Tekan tombol kombinasi Ctrl+Shift+Esc untuk membuka Task Manager.\n2. Pastikan tidak ada transaksi menggantung di mesin EDC.\n3. End Task proses FainayaPOS.exe.\n4. Buka Fainaya Store Support -> Pantau Windows Service -> Pastikan service MSSQL$POSDB berstatus Running.\n5. Buka kembali Fainaya POS dan lakukan re-login kasir.",
		},
		{
			"Pemecahan Masalah Barcode Scanner Tidak Terdeteksi",
			"Barcode Scanner",
			"Langkah pemulihan scanner USB mati atau tidak membaca barcode.",
			"1. Periksa kabel USB pada port belakang POS.\n2. Pindahkan ke port USB cadangan.\n3. Scan barcode factory reset pada buku panduan scanner.\n4. Tes buka Notepad dan scan salah satu barcode barang.",
		},
	}
	kbStmt, _ := db.Prepare("INSERT INTO knowledge_base (title, category, summary, content) VALUES (?, ?, ?, ?)")
	if kbStmt != nil {
		for _, k := range kbs {
			_, _ = kbStmt.Exec(k.title, k.cat, k.sum, k.content)
		}
		kbStmt.Close()
	}

	// 7. Initial Tickets
	tickets := []struct {
		tNo, cat, title, desc, urg, status, dev string
		repId                                  int
	}{
		{"TCK-202611-001", "Printer Struk Error", "Printer Kasir 1 cutter tidak memotong struk", "Struk tercetak normal tapi pisau auto-cutter tidak merespons", "High", "In Progress", "AST-PRN-01", 2},
		{"TCK-202611-002", "Kasir/POS Macet", "Aplikasi POS lambat saat proses diskon member", "Ada jeda 10 detik saat scan kartu member", "Medium", "Open", "AST-POS-02", 2},
	}
	tStmt, _ := db.Prepare("INSERT INTO tickets (ticket_no, category, title, description, urgency, status, device_name, reported_by_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
	if tStmt != nil {
		for _, t := range tickets {
			_, _ = tStmt.Exec(t.tNo, t.cat, t.title, t.desc, t.urg, t.status, t.dev, t.repId)
		}
		tStmt.Close()
	}

	// 8. Initial Vendor Contacts
	vendors := []struct {
		name, stype, pic, phone, email, sla string
	}{
		{"PT Bank Mandiri (EDC Support)", "Mesin EDC Merchant", "Helpdesk 24/7", "14000", "edc.support@mandiri.co.id", "SLA 4 Jam Penggantian Unit"},
		{"PT Epson Indonesia Support", "Hardware Printer POS", "Bpk. Hendra", "021-80866766", "support@epson.co.id", "SLA 1x24 Jam"},
		{"Telkom Indibiz Toko", "ISP Internet Utama", "Account Officer Andi", "147", "indibiz@telkom.co.id", "SLA 2 Jam Recovery"},
		{"HO IT Retail Fainaya", "Kantor Pusat IT Tier-2", "Tim Enterprise Support", "021-5551234", "ho.it@fainayastore.id", "SLA 1 Jam Respon"},
	}
	vStmt, _ := db.Prepare("INSERT INTO vendor_contacts (vendor_name, service_type, contact_person, phone, email, sla_response) VALUES (?, ?, ?, ?, ?, ?)")
	if vStmt != nil {
		for _, v := range vendors {
			_, _ = vStmt.Exec(v.name, v.stype, v.pic, v.phone, v.email, v.sla)
		}
		vStmt.Close()
	}

	// 9. Initial Software Catalog
	softwares := []struct {
		name, ver, cat, desc string
	}{
		{"Fainaya POS Client", "v3.4.2", "POS", "Aplikasi kasir retail terintegrasi"},
		{"Epson TM-T82X Advanced Printer Driver", "v5.12", "Driver", "Driver resmi printer struk kasir"},
		{"Honeywell Barcode Scanner Driver & Utility", "v2.1", "Driver", "Konfigurasi scanner virtual COM/HID"},
		{"Windows Defender Offline Definition Update", "2026.11.24", "Antivirus", "Pembaruan definisi virus luring tanpa koneksi internet"},
		{"WireGuard HO VPN Client", "v0.5.3", "Utility", "Koneksi VPN aman ke Head Office"},
	}
	swStmt, _ := db.Prepare("INSERT INTO software_catalog (name, version, category, description) VALUES (?, ?, ?, ?)")
	if swStmt != nil {
		for _, sw := range softwares {
			_, _ = swStmt.Exec(sw.name, sw.ver, sw.cat, sw.desc)
		}
		swStmt.Close()
	}

	log.Println("Database seeded successfully.")
}
