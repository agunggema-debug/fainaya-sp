
  // State
  let user: User | null = null;
  let activeTab: string = 'dashboard';
  let currentTheme: string = localStorage.getItem('fainaya_theme') || 'daylight-crisp';

  // Auth form
  let loginUsername = '';
  let loginPassword = '';
  let loginError = '';
  let loginLoading = false;

  // Data states
  let tickets: Ticket[] = [];
  let checklist: ChecklistItem[] = [];
  let assets: Asset[] = [];
  let softwares: Software[] = [];
  let antivirusList: AntivirusStatus[] = [];
  let networkDevices: NetworkDevice[] = [];
  let serverHealth: ServerHealth | null = null;
  let windowsServices: WindowsService[] = [];
  let activityLogs: ActivityLog[] = [];
  let knowledgeBases: KnowledgeBase[] = [];
  let vendors: VendorContact[] = [];

  // Modals
  let showCreateTicketModal = false;
  let showEscalateModal = false;
  let selectedTicketId: number | null = null;
  let newTicketCategory = 'Kasir/POS Macet';
  let newTicketTitle = '';
  let newTicketDesc = '';
  let newTicketUrgency = 'High';
  let newTicketDevice = 'AST-POS-01';

  let escalateTarget = 'HO IT Team';
  let escalateNotes = '';

  // Network diagnostic
  let diagTarget = '192.168.10.1';
  let diagCommand = 'ping';
  let diagPort = '80';
  let diagResult: { success: boolean; output: string; duration_ms: number } | null = null;
  let diagRunning = false;

  // Themes list
  const themes = [
    { id: 'daylight-crisp', name: 'Daylight Crisp', icon: '☀️' },
    { id: 'midnight-noc', name: 'Midnight NOC', icon: '🌙' },
    { id: 'high-contrast-pos', name: 'High-Contrast POS', icon: '⚡' },
    { id: 'nordic-slate', name: 'Nordic Slate', icon: '❄️' },
    { id: 'warm-parchment', name: 'Warm Parchment', icon: '📜' },
    { id: 'terminal-ops', name: 'Terminal Ops', icon: '💻' }
  ];

  function applyTheme(themeId: string) {
    currentTheme = themeId;
    document.documentElement.setAttribute('data-theme', themeId);
    localStorage.setItem('fainaya_theme', themeId);
    if (user) {
      apiFetch('/theme', {
        method: 'PATCH',
        body: JSON.stringify({ theme: themeId })
      }).catch(() => {});
    }
  }

  onMount(async () => {
    applyTheme(currentTheme);
    const token = getToken();
    if (token) {
      try {
        user = await apiFetch('/me');
        if (user && user.theme_pref) {
          applyTheme(user.theme_pref);
        }
        loadAllData();
      } catch (e) {
        removeToken();
        user = null;
      }
    }
  });

  async function handleLogin() {
    loginError = '';
    loginLoading = true;
    try {
      const res = await apiFetch('/login', {
        method: 'POST',
        body: JSON.stringify({ username: loginUsername, password: loginPassword })
      });
      setToken(res.token);
      user = res.user;
      if (user && user.theme_pref) {
        applyTheme(user.theme_pref);
      }
      loadAllData();
    } catch (err: any) {
      loginError = err.message || 'Login gagal';
    } finally {
      loginLoading = false;
    }
  }

  function handleLogout() {
    removeToken();
    user = null;
  }

  async function loadAllData() {
    if (!user) return;
    try {
      tickets = await apiFetch('/tickets');
      checklist = await apiFetch('/checklists');
      assets = await apiFetch('/assets');
      softwares = await apiFetch('/software');
      antivirusList = await apiFetch('/antivirus');
      networkDevices = await apiFetch('/network/devices');
      serverHealth = await apiFetch('/server/health');
      windowsServices = await apiFetch('/server/services');
      knowledgeBases = await apiFetch('/knowledge-base');
      vendors = await apiFetch('/vendors');
      if (user.role === 'it_support' || user.role === 'store_manager' || user.role === 'ho_it' || user.role === 'super_admin') {
        activityLogs = await apiFetch('/logs');
      }
    } catch (e) {
      console.error('Error fetching data:', e);
    }
  }

  async function createTicket() {
    try {
      await apiFetch('/tickets', {
        method: 'POST',
        body: JSON.stringify({
          category: newTicketCategory,
          title: newTicketTitle,
          description: newTicketDesc,
          urgency: newTicketUrgency,
          device_name: newTicketDevice
        })
      });
      showCreateTicketModal = false;
      newTicketTitle = '';
      newTicketDesc = '';
      tickets = await apiFetch('/tickets');
    } catch (e: any) {
      alert(e.message);
    }
  }

  async function updateTicketStatus(id: number, status: string) {
    try {
      await apiFetch(`/tickets/${id}/status`, {
        method: 'PATCH',
        body: JSON.stringify({ status })
      });
      tickets = await apiFetch('/tickets');
    } catch (e: any) {
      alert(e.message);
    }
  }

  async function escalateTicket() {
    if (!selectedTicketId) return;
    try {
      await apiFetch(`/tickets/${selectedTicketId}/escalate`, {
        method: 'POST',
        body: JSON.stringify({
          escalated_to: escalateTarget,
          notes: escalateNotes
        })
      });
      showEscalateModal = false;
      escalateNotes = '';
      selectedTicketId = null;
      tickets = await apiFetch('/tickets');
    } catch (e: any) {
      alert(e.message);
    }
  }

  async function toggleCheck(item: ChecklistItem) {
    const nextVal = item.is_checked === 1 ? 0 : 1;
    try {
      await apiFetch(`/checklists/${item.id}/toggle`, {
        method: 'POST',
        body: JSON.stringify({ is_checked: nextVal, notes: item.notes })
      });
      checklist = await apiFetch('/checklists');
    } catch (e: any) {
      alert(e.message);
    }
  }

  async function runDiagnostic() {
    diagRunning = true;
    diagResult = null;
    try {
      const res = await apiFetch('/network/diagnostics', {
        method: 'POST',
        body: JSON.stringify({
          target: diagTarget,
          command: diagCommand,
          port: diagPort
        })
      });
      diagResult = res;
    } catch (e: any) {
      diagResult = { success: false, output: e.message, duration_ms: 0 };
    } finally {
      diagRunning = false;
    }
  }

  async function controlService(serviceName: string, action: string) {
    try {
      await apiFetch('/server/services/control', {
        method: 'POST',
        body: JSON.stringify({ service_name: serviceName, action })
      });
      windowsServices = await apiFetch('/server/services');
    } catch (e: any) {
      alert(e.message);
    }
  }
</script>

{#if !user}
  <!-- LOGIN VIEW -->
  <div style="min-height: 100vh; display: flex; align-items: center; justify-content: center; background-color: var(--bg-canvas);">
    <div class="card" style="width: 100%; max-width: 440px;">
      <div style="text-align: center; margin-bottom: 24px;">
        <div style="background-color: var(--status-danger-bg); color: var(--status-danger-text); padding: 10px 14px; border-radius: 6px; font-size: 13px; margin-bottom: 16px;">
          ⚠️ {loginError}
        </div>
      {/if}

      <form on:submit|preventDefault={handleLogin}>
        <div>
          <label for="u">Username</label>
          <input id="u" type="text" bind:value={loginUsername} placeholder="e.g. admin, it_support, kasir1" required />
        </div>
        <div>
          <label for="p">Password</label>
          <input id="p" type="password" bind:value={loginPassword} placeholder="••••••••" required />
        </div>
        <button type="submit" class="btn btn-primary" style="width: 100%; margin-top: 12px;" disabled={loginLoading}>
          {loginLoading ? 'Memproses...' : 'Masuk ke Sistem Toko'}
        </button>
      </form>

      <div style="margin-top: 20px; font-size: 12px; color: var(--text-muted); border-top: 1px solid var(--border-subtle); padding-top: 16px;">
        <p style="font-weight: 600; margin-bottom: 4px;">Akun Bawaan Demo:</p>
        <p>• Admin: <code>admin / admin123</code></p>
        <p>• IT Support: <code>it_support / support123</code></p>
        <p>• Kasir 1: <code>kasir1 / kasir123</code></p>
        <p>• Store Mgr: <code>store_mgr / mgr123</code></p>
        <p>• HO IT: <code>ho_it / ho123</code></p>
      </div>
    </div>
  </div>
{:else}
  <!-- MAIN APP WITH 6-THEME ENGINE -->
  <div class="app-container">
    <!-- SIDEBAR -->
    <aside class="sidebar">
      <div style="padding: 20px; border-bottom: 1px solid var(--border-subtle);">
        <h2 style="font-size: 18px; font-weight: bold; color: var(--accent-primary);">FAINAYA IT</h2>
        <span class="badge badge-success" style="margin-top: 4px;">{user.store_code} • ONLINE</span>
      </div>

          📋 Checklist Toko
        </button>

        <button class="btn" style="justify-content: flex-start; {activeTab === 'hardware' ? 'background: var(--accent-subtle); color: var(--accent-primary); font-weight: bold;' : 'background: transparent; color: var(--text-secondary);'}" on:click={() => activeTab = 'hardware'}>
          🖥️ Hardware & Aset
        </button>

        <button class="btn" style="justify-content: flex-start; {activeTab === 'software' ? 'background: var(--accent-subtle); color: var(--accent-primary); font-weight: bold;' : 'background: transparent; color: var(--text-secondary);'}" on:click={() => activeTab = 'software'}>
          🛡️ Software & Antivirus
        </button>
        <button class="btn" style="justify-content: flex-start; {activeTab === 'network' ? 'background: var(--accent-subtle); color: var(--accent-primary); font-weight: bold;' : 'background: transparent; color: var(--text-secondary);'}" on:click={() => activeTab = 'network'}>
          🌐 Jaringan (LAN/Wi-Fi)
        </button>

        <button class="btn" style="justify-content: flex-start; {activeTab === 'server' ? 'background: var(--accent-subtle); color: var(--accent-primary); font-weight: bold;' : 'background: transparent; color: var(--text-secondary);'}" on:click={() => activeTab = 'server'}>
        <div style="font-size: 11px; color: var(--text-muted); text-transform: uppercase;">Peran: {user.role}</div>
        <button class="btn btn-outline" style="width: 100%; margin-top: 10px; font-size: 12px; padding: 4px;" on:click={handleLogout}>
          Keluar
        </button>
      </div>
    </aside>

    <!-- MAIN BODY -->
    <main class="main-content">
      <!-- TOP BAR WITH 6-THEME SELECTOR -->
      <header class="top-bar">
        <div style="display: flex; align-items: center; gap: 12px;">
          <h2 style="font-size: 16px; font-weight: 600; text-transform: capitalize;">
            {activeTab.replace('-', ' ')}
          </h2>
          <span style="font-size: 12px; color: var(--text-muted);">
            Intranet: <code>192.168.10.10:8080</code>
          </span>
        </div>

        <div style="display: flex; align-items: center; gap: 16px;">
          <!-- 6-THEME SWITCHER DROPDOWN (Zero Page Reload) -->
          <div style="display: flex; align-items: center; gap: 6px;">
            <span style="font-size: 12px; font-weight: 500; color: var(--text-secondary);">🎨 Tema:</span>
            <select
              style="width: auto; margin: 0; padding: 4px 8px; font-size: 12px;"
              bind:value={currentTheme}
              on:change={(e) => applyTheme(e.currentTarget.value)}
            >
              {#each themes as t}
                <option value={t.id}>{t.icon} {t.name}</option>
              {/each}
            </select>
          </div>

          <button class="btn btn-primary" on:click={() => showCreateTicketModal = true}>
            ➕ Buat Tiket Bantuan
          </button>
        </div>
      </header>

      <!-- CONTENT PANES -->
      <div class="content-body">
        {#if activeTab === 'dashboard'}
          <!-- QUICK STATS -->
          <div class="grid-3" style="margin-bottom: 24px;">
            <div class="card">
              <div style="font-size: 13px; color: var(--text-muted);">Total Tiket Masuk</div>
              <div style="font-size: 28px; font-weight: bold; margin-top: 4px; color: var(--accent-primary);">{tickets.length}</div>
              <div style="font-size: 12px; color: var(--text-secondary); margin-top: 4px;">
                {tickets.filter(t => t.status === 'Open').length} Perlu Penanganan
              </div>
            </div>

            <div class="card">
              <div style="font-size: 13px; color: var(--text-muted);">Kesehatan Windows Server</div>
              <div style="font-size: 28px; font-weight: bold; margin-top: 4px; color: #16a34a;">
                {serverHealth ? `${serverHealth.cpu_usage_pct}% CPU` : '99.8%'}
              </div>
              <div style="font-size: 12px; color: var(--text-secondary); margin-top: 4px;">
                RAM Terpakai: {serverHealth?.ram_used_gb} GB / {serverHealth?.ram_total_gb} GB
              </div>
            </div>

            <div class="card">
              <div style="font-size: 13px; color: var(--text-muted);">Status Jaringan Kasir & Gateway</div>
              <div style="font-size: 28px; font-weight: bold; margin-top: 4px; color: #16a34a;">
                {networkDevices.filter(d => d.status === 'Online').length} / {networkDevices.length} Online
              </div>
              <div style="font-size: 12px; color: var(--text-secondary); margin-top: 4px;">
                Semua Jalur Kasir & EDC Terhubung
              </div>
            </div>
          </div>

          <!-- TICKETS LIST -->
          <div class="card">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px;">
              <h3 style="font-size: 16px; font-weight: 600;">Daftar Tiket Kendala Operasional Toko</h3>
              <button class="btn btn-outline" style="font-size: 12px;" on:click={loadAllData}>Segarkan Data</button>
            </div>

            <table>
              <thead>
                <tr>
                  <th>No Tiket</th>
                  <th>Kategori</th>
                  <th>Judul Masalah</th>
                  <th>Urgensi</th>
                  <th>Perangkat</th>
                  <th>Pelapor</th>
                  <th>Status</th>
                  <th>Aksi</th>
                </tr>
              </thead>
              <tbody>
                {#each tickets as t}
                  <tr>
                    <td><code>{t.ticket_no}</code></td>
                    <td>{t.category}</td>
                    <td>
                      <div style="font-weight: 500;">{t.title}</div>
                      <div style="font-size: 12px; color: var(--text-muted);">{t.description}</div>
                      {#if t.escalated_to}
                        <div style="font-size: 11px; color: #dc2626; font-weight: 600; margin-top: 2px;">
                          🚀 Dieskalasi ke: {t.escalated_to} ({t.escalation_notes})
                        </div>
                      {/if}
                    </td>
                    <td>
                      <span class="badge {t.urgency === 'Critical' || t.urgency === 'High' ? 'badge-danger' : 'badge-warning'}">
                        {t.urgency}
                      </span>
                    </td>
                    <td>{t.device_name || '-'}</td>
                    <td>{t.reported_by_name}</td>
                    <td>
                      <span class="badge {t.status === 'Resolved' ? 'badge-success' : t.status === 'Escalated' ? 'badge-danger' : 'badge-warning'}">
                        {t.status}
                      </span>
                    </td>
                    <td>
                      <div style="display: flex; gap: 6px;">
                        {#if t.status !== 'Resolved' && (user.role === 'it_support' || user.role === 'store_manager' || user.role === 'super_admin')}
                          <button class="btn btn-outline" style="padding: 2px 8px; font-size: 11px;" on:click={() => updateTicketStatus(t.id, 'Resolved')}>
                            Selesai
                          </button>
                          <button class="btn btn-outline" style="padding: 2px 8px; font-size: 11px; color: #dc2626;" on:click={() => { selectedTicketId = t.id; showEscalateModal = true; }}>
                            Eskalasi
                          </button>
                        {/if}
                      </div>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>

        {:else if activeTab === 'checklist'}
          <!-- DAILY IT READINESS CHECKLIST -->
          <div class="card">
            <h3 style="font-size: 16px; font-weight: 600; margin-bottom: 8px;">Checklist Kesiapan TI Toko (Daily IT Readiness)</h3>
            <p style="font-size: 13px; color: var(--text-secondary); margin-bottom: 16px;">
              Memastikan POS, EDC, Jaringan, dan Server siap sebelum buka (Opening) dan aman saat tutup (Closing).
            </p>

            <div style="display: flex; flex-direction: column; gap: 8px;">
              {#each checklist as item}
                <div style="display: flex; align-items: center; justify-content: space-between; padding: 12px; background: var(--bg-surface-elevated); border-radius: 6px; border: 1px solid var(--border-subtle);">
                  <div style="display: flex; align-items: center; gap: 12px;">
                    <input
                      type="checkbox"
                      checked={item.is_checked === 1}
                      style="width: 20px; height: 20px; margin: 0; cursor: pointer;"
                      on:change={() => toggleCheck(item)}
                      disabled={user.role !== 'it_support' && user.role !== 'super_admin'}
                    />
                    <div>
                      <div style="font-weight: 600; {item.is_checked === 1 ? 'text-decoration: line-through; opacity: 0.6;' : ''}">
                        {item.item_name}
                      </div>
                      <span class="badge" style="font-size: 10px; background: var(--accent-subtle); color: var(--accent-primary);">
                        {item.category} • {item.shift_type}
                      </span>
                    </div>
                  </div>
                  <div>
                    <span class="badge {item.is_checked === 1 ? 'badge-success' : 'badge-danger'}">
                      {item.is_checked === 1 ? 'Sudah Diperiksa' : 'Belum'}
                    </span>
                  </div>
                </div>
              {/each}
            </div>
          </div>

        {:else if activeTab === 'hardware'}
          <!-- HARDWARE ASSET INVENTORY -->
          <div class="card">
            <h3 style="font-size: 16px; font-weight: 600; margin-bottom: 16px;">Registri Aset Hardware & Jadwal Maintenance</h3>
            <table>
              <thead>
                <tr>
                  <th>Kode Aset</th>
                  <th>Nama Perangkat</th>
                  <th>Kategori</th>
                  <th>Nomor Seri</th>
                  <th>Lokasi</th>
                  <th>Spesifikasi</th>
                  <th>Garansi</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {#each assets as a}
                  <tr>
                    <td><code>{a.asset_code}</code></td>
                    <td style="font-weight: 500;">{a.name}</td>
                    <td>{a.category}</td>
                    <td><code>{a.serial_number || '-'}</code></td>
                    <td>{a.location}</td>
                    <td style="font-size: 12px; color: var(--text-secondary);">{a.specs || '-'}</td>
                    <td style="font-size: 12px;">{a.warranty_until || '-'}</td>
                    <td><span class="badge badge-success">{a.status}</span></td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>

        {:else if activeTab === 'software'}
          <!-- SOFTWARE CATALOG & ANTIVIRUS HEALTH TRACKER -->
          <div class="grid-2">
            <div class="card">
              <h3 style="font-size: 16px; font-weight: 600; margin-bottom: 12px;">🛡️ Status Kepatuhan Antivirus Toko</h3>
              <table>
                <thead>
                  <tr>
                    <th>Terminal</th>
                    <th>Definisi Terakhir</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {#each antivirusList as av}
                    <tr>
                      <td>
                        <div style="font-weight: 600;">{av.hostname}</div>
                        <div style="font-size: 11px; color: var(--text-muted);">{av.ip_address}</div>
                      </td>
                      <td>
                        <div>{av.defs_updated_at}</div>
                        <div style="font-size: 11px; color: var(--text-muted);">Scan: {av.last_scan_at}</div>
                      </td>
                      <td>
                        <span class="badge {av.status === 'Healthy' ? 'badge-success' : 'badge-warning'}">
                          {av.status}
                        </span>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>

            <div class="card">
              <h3 style="font-size: 16px; font-weight: 600; margin-bottom: 12px;">📦 Katalog & Installer Standar Toko</h3>
              <table>
                <thead>
                  <tr>
                    <th>Software</th>
                    <th>Versi</th>
                    <th>Kategori</th>
                  </tr>
                </thead>
                <tbody>
                  {#each softwares as sw}
                    <tr>
                      <td>
                        <div style="font-weight: 600;">{sw.name}</div>
                        <div style="font-size: 11px; color: var(--text-muted);">{sw.description}</div>
                      </td>
                      <td><code>{sw.version}</code></td>
                      <td><span class="badge" style="background: var(--accent-subtle); color: var(--accent-primary);">{sw.category}</span></td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          </div>

        {:else if activeTab === 'network'}
          <!-- NETWORK MONITOR & DIAGNOSTIC UTILITY -->
          <div class="grid-2">
            <div class="card">
              <h3 style="font-size: 16px; font-weight: 600; margin-bottom: 12px;">Perangkat Jaringan LAN & Wi-Fi</h3>
              <table>
                <thead>
                  <tr>
                    <th>Perangkat</th>
                    <th>IP Address</th>
                    <th>VLAN</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {#each networkDevices as dev}
                    <tr>
                      <td style="font-weight: 500;">{dev.name}</td>
                      <td><code>{dev.ip_address}</code></td>
                      <td>{dev.vlan || '-'}</td>
                      <td>
                        <span class="badge {dev.status === 'Online' ? 'badge-success' : 'badge-danger'}">
                          {dev.status} ({dev.last_ping_ms}ms)
                        </span>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>

            <!-- DIAGNOSTIC UTILITIES -->
            <div class="card">
              <h3 style="font-size: 16px; font-weight: 600; margin-bottom: 12px;">⚡ Alat Diagnostik Jaringan Toko</h3>
              <div style="margin-bottom: 12px;">
                <label for="dt">Target IP / Host</label>
                <input id="dt" type="text" bind:value={diagTarget} placeholder="192.168.10.1" />
              </div>

              <div style="display: flex; gap: 10px; margin-bottom: 12px;">
                <div style="flex: 1;">
                  <label for="dc">Perintah</label>
                  <select id="dc" bind:value={diagCommand}>
                    <option value="ping">Ping Jaringan</option>
                    <option value="dns">DNS Lookup</option>
                    <option value="tcp_check">Pengecekan Port TCP</option>
                  </select>
                </div>

                {#if diagCommand === 'tcp_check'}
                  <div style="width: 100px;">
                    <label for="dp">Port</label>
                    <input id="dp" type="text" bind:value={diagPort} placeholder="80" />
                  </div>
                {/if}
              </div>

              <button class="btn btn-primary" style="width: 100%;" on:click={runDiagnostic} disabled={diagRunning}>
                {diagRunning ? 'Menjalankan Diagnostik...' : 'Jalankan Pengujian'}
              </button>

              {#if diagResult}
                <div style="margin-top: 16px; background: var(--bg-surface-elevated); padding: 12px; border-radius: 6px; border: 1px solid var(--border-subtle);">
                  <div style="font-size: 12px; font-weight: 600; margin-bottom: 6px;">
                    Hasil Diagnostik ({diagResult.duration_ms} ms):
                  </div>
                  <pre style="font-size: 11px; white-space: pre-wrap; font-family: monospace; color: var(--text-primary);">{diagResult.output}</pre>
                </div>
              {/if}
            </div>
          </div>

        {:else if activeTab === 'server'}
          <!-- WINDOWS SERVER DASHBOARD & SERVICE CONTROLS -->
          <div class="grid-2">
            <div class="card">
              <h3 style="font-size: 16px; font-weight: 600; margin-bottom: 12px;">Ringkasan Kesehatan Server Toko</h3>
              {#if serverHealth}
                <div style="display: flex; flex-direction: column; gap: 12px;">
                  <div>
                    <div style="display: flex; justify-content: space-between; font-size: 13px;">
                      <span>Beban CPU</span>
                      <strong>{serverHealth.cpu_usage_pct}%</strong>
                    </div>
                    <div style="height: 8px; background: var(--bg-surface-elevated); border-radius: 4px; overflow: hidden; margin-top: 4px;">
                      <div style="width: {serverHealth.cpu_usage_pct}%; height: 100%; background: #0284c7;"></div>
                    </div>
                  </div>

                  <div>
                    <div style="display: flex; justify-content: space-between; font-size: 13px;">
                      <span>Penggunaan Memori (RAM)</span>
                      <strong>{serverHealth.ram_usage_pct}% ({serverHealth.ram_used_gb} GB / {serverHealth.ram_total_gb} GB)</strong>
                    </div>
                    <div style="height: 8px; background: var(--bg-surface-elevated); border-radius: 4px; overflow: hidden; margin-top: 4px;">
                      <div style="width: {serverHealth.ram_usage_pct}%; height: 100%; background: #16a34a;"></div>
                    </div>
                  </div>

                  {#each serverHealth.disks as disk}
                    <div>
                      <div style="display: flex; justify-content: space-between; font-size: 13px;">
                        <span>Partisi {disk.drive}</span>
                        <strong>Sisa {disk.free_gb} GB ({disk.free_pct}%)</strong>
                      </div>
                      <div style="height: 8px; background: var(--bg-surface-elevated); border-radius: 4px; overflow: hidden; margin-top: 4px;">
                        <div style="width: {100 - disk.free_pct}%; height: 100%; background: #818cf8;"></div>
                      </div>
                    </div>
                  {/each}

                  <div style="padding: 10px; background: var(--bg-surface-elevated); border-radius: 6px; font-size: 12px;">
                    <div><strong>Uptime:</strong> {serverHealth.uptime_days} Hari Aktif</div>
                    <div><strong>Backup Terakhir:</strong> {serverHealth.last_backup_at}</div>
                    <div><strong>Status Backup:</strong> <span class="badge badge-success">{serverHealth.backup_status}</span></div>
                  </div>
                </div>
              {/if}
            </div>

            <!-- WINDOWS SERVICES CONTROLLER -->
            <div class="card">
              <h3 style="font-size: 16px; font-weight: 600; margin-bottom: 12px;">Layanan Windows Server (Whitelisted)</h3>
              <p style="font-size: 12px; color: var(--text-secondary); margin-bottom: 12px;">
                Kontrol aman terhadap service resmi tanpa resiko kesalahan sistem operasi.
              </p>
              <table>
                <thead>
                  <tr>
                    <th>Nama Layanan</th>
                    <th>Status</th>
                    <th>Aksi</th>
                  </tr>
                </thead>
                <tbody>
                  {#each windowsServices as ws}
                    <tr>
                      <td>
                        <div style="font-weight: 600;">{ws.display_name}</div>
                        <div style="font-size: 11px; color: var(--text-muted); font-family: monospace;">{ws.service_name}</div>
                      </td>
                      <td>
                        <span class="badge {ws.status === 'Running' ? 'badge-success' : 'badge-danger'}">
                          {ws.status}
                        </span>
                      </td>
                      <td>
                        {#if user.role === 'it_support' || user.role === 'ho_it' || user.role === 'super_admin'}
                          <button class="btn btn-outline" style="padding: 4px 8px; font-size: 11px;" on:click={() => controlService(ws.service_name, 'Restart')}>
                            Restart
                          </button>
                        {/if}
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          </div>

        {:else if activeTab === 'kb'}
          <!-- KNOWLEDGE BASE & SOP -->
          <div class="card">
            <h3 style="font-size: 16px; font-weight: 600; margin-bottom: 16px;">Basis Pengetahuan & SOP Penanganan Kendala Toko</h3>
            <div style="display: flex; flex-direction: column; gap: 16px;">
              {#each knowledgeBases as kb}
                <div style="border: 1px solid var(--border-subtle); border-radius: 8px; padding: 16px; background: var(--bg-surface-elevated);">
                  <div style="display: flex; justify-content: space-between; align-items: center;">
                    <h4 style="font-size: 15px; font-weight: 600; color: var(--accent-primary);">{kb.title}</h4>
                    <span class="badge" style="background: var(--accent-subtle); color: var(--accent-primary);">{kb.category}</span>
                  </div>
                  <p style="font-size: 13px; color: var(--text-secondary); margin: 6px 0 10px 0;">{kb.summary}</p>
                  <pre style="background: var(--bg-surface); padding: 12px; border-radius: 6px; font-size: 12px; white-space: pre-wrap; font-family: inherit; border: 1px solid var(--border-subtle);">{kb.content}</pre>
                </div>
              {/each}
            </div>
          </div>

        {:else if activeTab === 'vendors'}
          <!-- VENDOR SLA & HO CONTACT DIRECTORY -->
          <div class="card">
            <h3 style="font-size: 16px; font-weight: 600; margin-bottom: 16px;">Direktori Kontak & Target SLA Vendor / Head Office</h3>
            <table>
              <thead>
                <tr>
                  <th>Penyedia / Vendor</th>
                  <th>Layanan</th>
                  <th>Kontak PIC</th>
                  <th>Target SLA</th>
                </tr>
              </thead>
              <tbody>
                {#each vendors as v}
                  <tr>
                    <td style="font-weight: 600;">{v.vendor_name}</td>
                    <td>{v.service_type}</td>
                    <td>
                      <div>📞 {v.phone}</div>
                      {#if v.email}<div style="font-size: 11px; color: var(--text-muted);">{v.email}</div>{/if}
                    </td>
                    <td><span class="badge badge-warning">{v.sla_response}</span></td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>

        {:else if activeTab === 'logs'}
          <!-- IMMUTABLE AUDIT TRAIL -->
          <div class="card">
            <h3 style="font-size: 16px; font-weight: 600; margin-bottom: 16px;">Jejak Audit Aktivitas Sistem (Immutable Audit Trail)</h3>
            <table>
              <thead>
                <tr>
                  <th>Waktu</th>
                  <th>Pengguna</th>
                  <th>Aktivitas</th>
                  <th>Keterangan</th>
                  <th>IP Address</th>
                </tr>
              </thead>
              <tbody>
                {#each activityLogs as l}
                  <tr>
                    <td style="font-size: 12px; white-space: nowrap;">{l.created_at}</td>
                    <td style="font-weight: 600;">{l.username}</td>
                    <td><code>{l.activity_type}</code></td>
                    <td style="font-size: 13px;">{l.description}</td>
                    <td><code style="font-size: 11px;">{l.ip_address || '-'}</code></td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>
    </main>
  </div>

  <!-- MODAL: CREATE TICKET -->
  {#if showCreateTicketModal}
    <div class="modal-overlay" on:click|self={() => showCreateTicketModal = false}>
      <div class="modal-content">
        <h3 style="font-size: 18px; font-weight: 600; margin-bottom: 16px;">Buat Tiket Bantuan Cepat</h3>
        
        <div>
          <label for="tc">Kategori Kendala</label>
          <select id="tc" bind:value={newTicketCategory}>
            <option value="Kasir/POS Macet">Kasir/POS Macet</option>
            <option value="Printer Struk Error">Printer Struk Error</option>
            <option value="Barcode Scanner Mati">Barcode Scanner Mati</option>
            <option value="Internet/Wi-Fi Putus">Internet/Wi-Fi Putus</option>
            <option value="Aplikasi Error">Aplikasi Error</option>
            <option value="Hardware / UPS Rusak">Hardware / UPS Rusak</option>
          </select>
        </div>

        <div>
          <label for="tt">Judul Masalah</label>
          <input id="tt" type="text" bind:value={newTicketTitle} placeholder="Contoh: Kertas printer kasir 1 sering jam" required />
        </div>

        <div>
          <label for="td">Deskripsi Rinci</label>
          <textarea id="td" rows="3" bind:value={newTicketDesc} placeholder="Jelaskan detail kendala atau gejala yang muncul"></textarea>
        </div>

        <div style="display: flex; gap: 12px;">
          <div style="flex: 1;">
            <label for="tu">Tingkat Urgensi</label>
            <select id="tu" bind:value={newTicketUrgency}>
              <option value="Low">Low (Tidak mengganggu kasir)</option>
              <option value="Medium">Medium (Bisa alternatif)</option>
              <option value="High">High (Kasir terhambat)</option>
              <option value="Critical">Critical (Jalur Kasir Mati Total)</option>
            </select>
          <label for="en">Catatan Tambahan untuk HO / Vendor</label>

        <div style="display: flex; justify-content: flex-end; gap: 10px; margin-top: 16px;">
          <button class="btn btn-outline" on:click={() => showEscalateModal = false}>Batal</button>
          <button class="btn btn-primary" style="background-color: #dc2626;" on:click={escalateTicket}>Kirim Eskalasi</button>
        </div>
      </div>
    </div>
  {/if}
{/if}
