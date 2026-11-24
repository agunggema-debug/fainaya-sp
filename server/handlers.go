package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	FullName  string `json:"full_name"`
	Role      string `json:"role"`
	StoreCode string `json:"store_code"`
	ThemePref string `json:"theme_pref"`
	CreatedAt string `json:"created_at"`
}

type Ticket struct {
	ID              int     `json:"id"`
	TicketNo        string  `json:"ticket_no"`
	Category        string  `json:"category"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	Urgency         string  `json:"urgency"`
	Status          string  `json:"status"`
	DeviceName      *string `json:"device_name"`
	ReportedByID    int     `json:"reported_by_id"`
	ReportedByName  string  `json:"reported_by_name"`
	AssignedToID    *int    `json:"assigned_to_id"`
	EscalatedTo     *string `json:"escalated_to"`
	EscalationNotes *string `json:"escalation_notes"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type ChecklistItem struct {
	ID          int     `json:"id"`
	ShiftType   string  `json:"shift_type"`
	CheckDate   string  `json:"check_date"`
	ItemName    string  `json:"item_name"`
	Category    string  `json:"category"`
	IsChecked   int     `json:"is_checked"`
	Notes       *string `json:"notes"`
	CheckedByID *int    `json:"checked_by_id"`
}

// Log audit trail
func logActivity(c echo.Context, actType, desc string) {
	claims, ok := c.Get("user").(*JWTCustomClaims)
	uid := 0
	username := "System"
	if ok && claims != nil {
		uid = claims.UserID
		username = claims.Username
	}
	ip := c.RealIP()
	_, _ = db.Exec("INSERT INTO activity_logs (user_id, username, activity_type, description, ip_address) VALUES (?, ?, ?, ?, ?)",
		uid, username, actType, desc, ip)
}

// Handlers

func handleLogin(c echo.Context) error {
	type LoginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var req LoginReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Format permintaan tidak valid"})
	}

	var user User
	var hash string
	err := db.QueryRow("SELECT id, username, password_hash, full_name, role, store_code, theme_pref, created_at FROM users WHERE username = ?", req.Username).
		Scan(&user.ID, &user.Username, &hash, &user.FullName, &user.Role, &user.StoreCode, &user.ThemePref, &user.CreatedAt)

	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Username atau kata sandi salah"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Username atau kata sandi salah"})
	}

	token, err := generateToken(user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal menghasilkan token sesi"})
	}

	// Set HttpOnly cookie for safety
	cookie := &http.Cookie{
		Name:     "fainaya_token",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)

	logActivity(c, "LOGIN", fmt.Sprintf("User %s (%s) logged in successfully", user.Username, user.Role))

	return c.JSON(http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

func handleGetMe(c echo.Context) error {
	claims := c.Get("user").(*JWTCustomClaims)
	var user User
	err := db.QueryRow("SELECT id, username, full_name, role, store_code, theme_pref, created_at FROM users WHERE id = ?", claims.UserID).
		Scan(&user.ID, &user.Username, &user.FullName, &user.Role, &user.StoreCode, &user.ThemePref, &user.CreatedAt)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Pengguna tidak ditemukan"})
	}
	return c.JSON(http.StatusOK, user)
}

func handleUpdateTheme(c echo.Context) error {
	claims := c.Get("user").(*JWTCustomClaims)
	type ThemeReq struct {
		Theme string `json:"theme"`
	}
	var req ThemeReq
	if err := c.Bind(&req); err != nil || req.Theme == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Tema tidak valid"})
	}

	_, err := db.Exec("UPDATE users SET theme_pref = ? WHERE id = ?", req.Theme, claims.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal menyimpan preferensi tema"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Preferensi tema berhasil diperbarui", "theme": req.Theme})
}

// Tickets
func handleGetTickets(c echo.Context) error {
	claims := c.Get("user").(*JWTCustomClaims)

	var rows *sql.Rows
	var err error

	// RBAC filter: store_user only sees own tickets, vendor sees assigned, others see all
	if claims.Role == "store_user" {
		rows, err = db.Query(`
			SELECT t.id, t.ticket_no, t.category, t.title, t.description, t.urgency, t.status, 
			       t.device_name, t.reported_by_id, u.full_name, t.assigned_to_id, t.escalated_to, 
			       t.escalation_notes, t.created_at, t.updated_at
			FROM tickets t
			JOIN users u ON t.reported_by_id = u.id
			WHERE t.reported_by_id = ?
			ORDER BY t.id DESC`, claims.UserID)
	} else if claims.Role == "vendor" {
		rows, err = db.Query(`
			SELECT t.id, t.ticket_no, t.category, t.title, t.description, t.urgency, t.status, 
			       t.device_name, t.reported_by_id, u.full_name, t.assigned_to_id, t.escalated_to, 
			       t.escalation_notes, t.created_at, t.updated_at
			FROM tickets t
			JOIN users u ON t.reported_by_id = u.id
			WHERE t.escalated_to = 'Vendor' OR t.assigned_to_id = ?
			ORDER BY t.id DESC`, claims.UserID)
	} else {
		rows, err = db.Query(`
			SELECT t.id, t.ticket_no, t.category, t.title, t.description, t.urgency, t.status, 
			       t.device_name, t.reported_by_id, u.full_name, t.assigned_to_id, t.escalated_to, 
			       t.escalation_notes, t.created_at, t.updated_at
			FROM tickets t
			JOIN users u ON t.reported_by_id = u.id
			ORDER BY t.id DESC`)
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal mengambil data tiket"})
	}
	defer rows.Close()

	tickets := []Ticket{}
	for rows.Next() {
		var t Ticket
		if err := rows.Scan(&t.ID, &t.TicketNo, &t.Category, &t.Title, &t.Description, &t.Urgency, &t.Status,
			&t.DeviceName, &t.ReportedByID, &t.ReportedByName, &t.AssignedToID, &t.EscalatedTo, &t.EscalationNotes,
			&t.CreatedAt, &t.UpdatedAt); err == nil {
			tickets = append(tickets, t)
		}
	}

	return c.JSON(http.StatusOK, tickets)
}

func handleCreateTicket(c echo.Context) error {
	claims := c.Get("user").(*JWTCustomClaims)

	type CreateTicketReq struct {
		Category   string  `json:"category"`
		Title      string  `json:"title"`
		Description string `json:"description"`
		Urgency    string  `json:"urgency"`
		DeviceName string  `json:"device_name"`
	}

	var req CreateTicketReq
	if err := c.Bind(&req); err != nil || req.Category == "" || req.Title == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Kategori dan judul tiket wajib diisi"})
	}

	if req.Urgency == "" {
		req.Urgency = "Medium"
	}

	ticketNo := fmt.Sprintf("TCK-%s-%04d", time.Now().Format("20060102"), rand.Intn(9999))

	res, err := db.Exec(`
		INSERT INTO tickets (ticket_no, category, title, description, urgency, status, device_name, reported_by_id)
		VALUES (?, ?, ?, ?, ?, 'Open', ?, ?)`,
		ticketNo, req.Category, req.Title, req.Description, req.Urgency, req.DeviceName, claims.UserID)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal menyimpan tiket"})
	}

	id, _ := res.LastInsertId()
	logActivity(c, "TICKET_CREATE", fmt.Sprintf("Created ticket #%s (%s)", ticketNo, req.Title))

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message":   "Tiket berhasil dibuat",
		"ticket_id": id,
		"ticket_no": ticketNo,
	})
}

func handleUpdateTicketStatus(c echo.Context) error {
	id := c.Param("id")
	type StatusReq struct {
		Status string `json:"status"`
	}
	var req StatusReq
	if err := c.Bind(&req); err != nil || req.Status == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Status tidak valid"})
	}

	_, err := db.Exec("UPDATE tickets SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", req.Status, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal memperbarui status tiket"})
	}

	logActivity(c, "TICKET_UPDATE", fmt.Sprintf("Updated ticket ID %s to status %s", id, req.Status))
	return c.JSON(http.StatusOK, map[string]string{"message": "Status tiket berhasil diperbarui"})
}

func handleEscalateTicket(c echo.Context) error {
	id := c.Param("id")
	type EscalateReq struct {
		EscalatedTo string `json:"escalated_to"` // 'HO IT Team' or 'Vendor'
		Notes       string `json:"notes"`
	}
	var req EscalateReq
	if err := c.Bind(&req); err != nil || req.EscalatedTo == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Tujuan eskalasi harus ditentukan"})
	}

	_, err := db.Exec("UPDATE tickets SET escalated_to = ?, escalation_notes = ?, status = 'Escalated', updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		req.EscalatedTo, req.Notes, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal mengeksekusi eskalasi tiket"})
	}

	logActivity(c, "TICKET_ESCALATE", fmt.Sprintf("Ticket ID %s escalated to %s", id, req.EscalatedTo))
	return c.JSON(http.StatusOK, map[string]string{"message": "Tiket berhasil dieskalasi"})
}

// Checklist Daily IT Readiness
func handleGetChecklist(c echo.Context) error {
	dateStr := c.QueryParam("date")
	shift := c.QueryParam("shift")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	if shift == "" {
		shift = "Opening"
	}

	rows, err := db.Query("SELECT id, shift_type, check_date, item_name, category, is_checked, notes, checked_by_id FROM checklists WHERE check_date = ? AND shift_type = ?", dateStr, shift)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal membaca checklist"})
	}
	defer rows.Close()

	items := []ChecklistItem{}
	for rows.Next() {
		var it ChecklistItem
		if err := rows.Scan(&it.ID, &it.ShiftType, &it.CheckDate, &it.ItemName, &it.Category, &it.IsChecked, &it.Notes, &it.CheckedByID); err == nil {
			items = append(items, it)
		}
	}

	// Auto-seed checklist for today if empty
	if len(items) == 0 {
		defaultChecks := []struct {
			item, cat string
		}{
			{"Koneksi LAN & Wi-Fi Toko Utama", "Jaringan"},
			{"Periksa Kertas & Auto-cutter Printer Kasir 1 & 2", "Printer"},
			{"Periksa Daya & Baterai UPS Ruang Kasir", "Hardware"},
			{"Verifikasi Service SQL Server POS Berjalan", "Server"},
			{"Tes Koneksi EDC Merchant ke Bank", "EDC/Periferal"},
			{"Status Antivirus Windows Defender Hijau", "Keamanan"},
		}
		for _, dc := range defaultChecks {
			res, _ := db.Exec("INSERT INTO checklists (shift_type, check_date, item_name, category, is_checked) VALUES (?, ?, ?, ?, 0)",
				shift, dateStr, dc.item, dc.cat)
			id, _ := res.LastInsertId()
			items = append(items, ChecklistItem{
				ID:        int(id),
				ShiftType: shift,
				CheckDate: dateStr,
				ItemName:  dc.item,
				Category:  dc.cat,
				IsChecked: 0,
			})
		}
	}

	return c.JSON(http.StatusOK, items)
}

func handleToggleChecklist(c echo.Context) error {
	claims := c.Get("user").(*JWTCustomClaims)
	id := c.Param("id")

	type ToggleReq struct {
		IsChecked int     `json:"is_checked"`
		Notes     *string `json:"notes"`
	}
	var req ToggleReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Format tidak valid"})
	}

	_, err := db.Exec("UPDATE checklists SET is_checked = ?, notes = ?, checked_by_id = ? WHERE id = ?",
		req.IsChecked, req.Notes, claims.UserID, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal memperbarui checklist"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Checklist berhasil diperbarui"})
}

// Assets & Maintenance
func handleGetAssets(c echo.Context) error {
	rows, err := db.Query("SELECT id, asset_code, name, category, serial_number, location, status, specs, warranty_until, last_maintenance FROM assets ORDER BY id ASC")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal membaca data aset"})
	}
	defer rows.Close()

	type Asset struct {
		ID              int     `json:"id"`
		AssetCode       string  `json:"asset_code"`
		Name            string  `json:"name"`
		Category        string  `json:"category"`
		SerialNumber    *string `json:"serial_number"`
		Location        string  `json:"location"`
		Status          string  `json:"status"`
		Specs           *string `json:"specs"`
		WarrantyUntil   *string `json:"warranty_until"`
		LastMaintenance *string `json:"last_maintenance"`
	}

	assets := []Asset{}
	for rows.Next() {
		var a Asset
		if err := rows.Scan(&a.ID, &a.AssetCode, &a.Name, &a.Category, &a.SerialNumber, &a.Location, &a.Status, &a.Specs, &a.WarrantyUntil, &a.LastMaintenance); err == nil {
			assets = append(assets, a)
		}
	}
	return c.JSON(http.StatusOK, assets)
}

// Software & Antivirus
func handleGetSoftware(c echo.Context) error {
	rows, err := db.Query("SELECT id, name, version, category, status, description FROM software_catalog")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal membaca katalog software"})
	}
	defer rows.Close()

	type Software struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Version     string `json:"version"`
		Category    string `json:"category"`
		Status      string `json:"status"`
		Description string `json:"description"`
	}
	list := []Software{}
	for rows.Next() {
		var s Software
		if err := rows.Scan(&s.ID, &s.Name, &s.Version, &s.Category, &s.Status, &s.Description); err == nil {
			list = append(list, s)
		}
	}
	return c.JSON(http.StatusOK, list)
}

func handleGetAntivirus(c echo.Context) error {
	rows, err := db.Query("SELECT id, hostname, ip_address, antivirus_name, service_active, defs_updated_at, last_scan_at, threats_found, status FROM antivirus_status")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal membaca status antivirus"})
	}
	defer rows.Close()

	type AV struct {
		ID            int    `json:"id"`
		Hostname      string `json:"hostname"`
		IPAddress     string `json:"ip_address"`
		AntivirusName string `json:"antivirus_name"`
		ServiceActive int    `json:"service_active"`
		DefsUpdatedAt string `json:"defs_updated_at"`
		LastScanAt    string `json:"last_scan_at"`
		ThreatsFound  int    `json:"threats_found"`
		Status        string `json:"status"`
	}
	list := []AV{}
	for rows.Next() {
		var a AV
		if err := rows.Scan(&a.ID, &a.Hostname, &a.IPAddress, &a.AntivirusName, &a.ServiceActive, &a.DefsUpdatedAt, &a.LastScanAt, &a.ThreatsFound, &a.Status); err == nil {
			list = append(list, a)
		}
	}
	return c.JSON(http.StatusOK, list)
}

// Network Diagnostics (Ping & Devices)
func handleGetNetworkDevices(c echo.Context) error {
	rows, err := db.Query("SELECT id, name, ip_address, device_type, vlan, mac_address, status, last_ping_ms, last_checked FROM network_devices")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal membaca perangkat jaringan"})
	}
	defer rows.Close()

	type NetDev struct {
		ID          int     `json:"id"`
		Name        string  `json:"name"`
		IPAddress   string  `json:"ip_address"`
		DeviceType  string  `json:"device_type"`
		VLAN        *string `json:"vlan"`
		MACAddress  *string `json:"mac_address"`
		Status      string  `json:"status"`
		LastPingMs  float64 `json:"last_ping_ms"`
		LastChecked string  `json:"last_checked"`
	}
	list := []NetDev{}
	for rows.Next() {
		var d NetDev
		if err := rows.Scan(&d.ID, &d.Name, &d.IPAddress, &d.DeviceType, &d.VLAN, &d.MACAddress, &d.Status, &d.LastPingMs, &d.LastChecked); err == nil {
			list = append(list, d)
		}
	}
	return c.JSON(http.StatusOK, list)
}

// Built-in safe ping diagnostic (safe against command injection)
func handleRunDiagnostic(c echo.Context) error {
	type DiagReq struct {
		Target  string `json:"target"`
		Command string `json:"command"` // 'ping', 'dns', 'tcp_check'
		Port    string `json:"port"`
	}

	var req DiagReq
	if err := c.Bind(&req); err != nil || req.Target == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Target diagnostik wajib diisi"})
	}

	// Sanitize target: allow only valid hostnames or IPv4
	target := strings.TrimSpace(req.Target)
	if strings.ContainsAny(target, "&|;`$><\n\r\"'") {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Karakter tidak diizinkan pada target diagnostik demi keamanan"})
	}

	start := time.Now()
	var output string
	var success bool

	switch req.Command {
	case "ping":
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("ping", "-n", "3", target)
		} else {
			cmd = exec.Command("ping", "-c", "3", target)
		}
		out, err := cmd.CombinedOutput()
		output = string(out)
		success = err == nil
	case "dns":
		ips, err := net.LookupHost(target)
		if err != nil {
			output = fmt.Sprintf("DNS Lookup failed: %v", err)
			success = false
		} else {
			output = fmt.Sprintf("DNS Resolved successfully for %s:\n%s", target, strings.Join(ips, "\n"))
			success = true
		}
	case "tcp_check":
		port := req.Port
		if port == "" {
			port = "80"
		}
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(target, port), 3*time.Second)
		if err != nil {
			output = fmt.Sprintf("Port %s on %s is UNREACHABLE / CLOSED: %v", port, target, err)
			success = false
		} else {
			conn.Close()
			output = fmt.Sprintf("Port %s on %s is OPEN and RESPONDING.", port, target)
			success = true
		}
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Perintah diagnostik tidak dikenal"})
	}

	duration := time.Since(start).Milliseconds()
	logActivity(c, "NETWORK_DIAG", fmt.Sprintf("Executed %s on target %s (Success: %v, %dms)", req.Command, target, success, duration))

	return c.JSON(http.StatusOK, map[string]interface{}{
		"target":      target,
		"command":     req.Command,
		"success":     success,
		"output":      output,
		"duration_ms": duration,
	})
}

// Windows Server Monitor & Whitelisted Services
func handleGetServerHealth(c echo.Context) error {
	// Sample mock/live readings for Windows Server host
	type DiskPart struct {
		Drive      string  `json:"drive"`
		TotalGB    float64 `json:"total_gb"`
		FreeGB     float64 `json:"free_gb"`
		FreePct    float64 `json:"free_pct"`
		StatusFlag string  `json:"status_flag"`
	}

	type ServerHealth struct {
		Hostname     string     `json:"hostname"`
		OS           string     `json:"os"`
		CPUUsagePct  float64    `json:"cpu_usage_pct"`
		RAMUsagePct  float64    `json:"ram_usage_pct"`
		RAMTotalGB   float64    `json:"ram_total_gb"`
		RAMUsedGB    float64    `json:"ram_used_gb"`
		UptimeDays   float64    `json:"uptime_days"`
		Disks        []DiskPart `json:"disks"`
		LastBackupAt string     `json:"last_backup_at"`
		BackupStatus string     `json:"backup_status"`
	}

	host, _ := os.Hostname()
	health := ServerHealth{
		Hostname:     host,
		OS:           runtime.GOOS + " " + runtime.GOARCH,
		CPUUsagePct:  24.5,
		RAMUsagePct:  42.8,
		RAMTotalGB:   16.0,
		RAMUsedGB:    6.85,
		UptimeDays:   14.2,
		LastBackupAt: time.Now().Format("2006-01-02 03:00:00 (Daily POS DB Snapshot)"),
		BackupStatus: "Verified Success",
		Disks: []DiskPart{
			{Drive: "C: (OS & Apps)", TotalGB: 250, FreeGB: 110, FreePct: 44.0, StatusFlag: "Healthy"},
			{Drive: "D: (POS Data & Backup)", TotalGB: 500, FreeGB: 380, FreePct: 76.0, StatusFlag: "Healthy"},
		},
	}

	return c.JSON(http.StatusOK, health)
}

func handleGetWindowsServices(c echo.Context) error {
	rows, err := db.Query("SELECT id, service_name, display_name, status, is_whitelisted FROM windows_services")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal membaca daftar service"})
	}
	defer rows.Close()

	type WinService struct {
		ID            int    `json:"id"`
		ServiceName   string `json:"service_name"`
		DisplayName   string `json:"display_name"`
		Status        string `json:"status"`
		IsWhitelisted int    `json:"is_whitelisted"`
	}
	services := []WinService{}
	for rows.Next() {
		var s WinService
		if err := rows.Scan(&s.ID, &s.ServiceName, &s.DisplayName, &s.Status, &s.IsWhitelisted); err == nil {
			services = append(services, s)
		}
	}
	return c.JSON(http.StatusOK, services)
}

func handleControlWindowsService(c echo.Context) error {
	type ControlReq struct {
		ServiceName string `json:"service_name"`
		Action      string `json:"action"` // 'Restart', 'Start', 'Stop'
	}

	var req ControlReq
	if err := c.Bind(&req); err != nil || req.ServiceName == "" || req.Action == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Service dan aksi harus ditentukan"})
	}

	// Security: Verify service is in whitelist
	var isWhitelisted int
	err := db.QueryRow("SELECT is_whitelisted FROM windows_services WHERE service_name = ?", req.ServiceName).Scan(&isWhitelisted)
	if err != nil || isWhitelisted != 1 {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Aksi ditolak: Layanan tidak terdaftar dalam whitelist keamanan"})
	}

	// Simulate service action and update status
	newStatus := "Running"
	if req.Action == "Stop" {
		newStatus = "Stopped"
	}
	_, _ = db.Exec("UPDATE windows_services SET status = ? WHERE service_name = ?", newStatus, req.ServiceName)

	logActivity(c, "SERVICE_CONTROL", fmt.Sprintf("Executed action %s on Windows Service %s (Authorized)", req.Action, req.ServiceName))

	return c.JSON(http.StatusOK, map[string]string{
		"message":      fmt.Sprintf("Aksi %s pada layanan %s berhasil dieksekusi", req.Action, req.ServiceName),
		"service_name": req.ServiceName,
		"new_status":   newStatus,
	})
}

// Activity Log & Knowledge Base & Vendors
func handleGetActivityLogs(c echo.Context) error {
	rows, err := db.Query("SELECT id, user_id, username, activity_type, description, ip_address, created_at FROM activity_logs ORDER BY id DESC LIMIT 50")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal membaca audit log"})
	}
	defer rows.Close()

	type ActLog struct {
		ID           int     `json:"id"`
		UserID       *int    `json:"user_id"`
		Username     string  `json:"username"`
		ActivityType string  `json:"activity_type"`
		Description  string  `json:"description"`
		IPAddress    *string `json:"ip_address"`
		CreatedAt    string  `json:"created_at"`
	}
	logs := []ActLog{}
	for rows.Next() {
		var l ActLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.Username, &l.ActivityType, &l.Description, &l.IPAddress, &l.CreatedAt); err == nil {
			logs = append(logs, l)
		}
	}
	return c.JSON(http.StatusOK, logs)
}

func handleGetKnowledgeBase(c echo.Context) error {
	rows, err := db.Query("SELECT id, title, category, summary, content, created_at FROM knowledge_base ORDER BY id ASC")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal membaca knowledge base"})
	}
	defer rows.Close()

	type KB struct {
		ID        int    `json:"id"`
		Title     string `json:"title"`
		Category  string `json:"category"`
		Summary   string `json:"summary"`
		Content   string `json:"content"`
		CreatedAt string `json:"created_at"`
	}
	list := []KB{}
	for rows.Next() {
		var k KB
		if err := rows.Scan(&k.ID, &k.Title, &k.Category, &k.Summary, &k.Content, &k.CreatedAt); err == nil {
			list = append(list, k)
		}
	}
	return c.JSON(http.StatusOK, list)
}

func handleGetVendors(c echo.Context) error {
	rows, err := db.Query("SELECT id, vendor_name, service_type, contact_person, phone, email, sla_response, contract_info FROM vendor_contacts")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal membaca kontak vendor"})
	}
	defer rows.Close()

	type Vendor struct {
		ID            int     `json:"id"`
		VendorName    string  `json:"vendor_name"`
		ServiceType   string  `json:"service_type"`
		ContactPerson *string `json:"contact_person"`
		Phone         string  `json:"phone"`
		Email         *string `json:"email"`
		SLAResponse   *string `json:"sla_response"`
		ContractInfo  *string `json:"contract_info"`
	}
	vendors := []Vendor{}
	for rows.Next() {
		var v Vendor
		if err := rows.Scan(&v.ID, &v.VendorName, &v.ServiceType, &v.ContactPerson, &v.Phone, &v.Email, &v.SLAResponse, &v.ContractInfo); err == nil {
			vendors = append(vendors, v)
		}
	}
	return c.JSON(http.StatusOK, vendors)
}

// Users management for super_admin
func handleGetUsers(c echo.Context) error {
	rows, err := db.Query("SELECT id, username, full_name, role, store_code, theme_pref, created_at FROM users")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Gagal membaca user"})
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.FullName, &u.Role, &u.StoreCode, &u.ThemePref, &u.CreatedAt); err == nil {
			users = append(users, u)
		}
	}
	return c.JSON(http.StatusOK, users)
}
