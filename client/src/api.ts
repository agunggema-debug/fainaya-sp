const API_BASE = '/api';

export function getToken(): string | null {
  return localStorage.getItem('fainaya_token');
}

export function setToken(token: string) {
  localStorage.setItem('fainaya_token', token);
}

export function removeToken() {
  localStorage.removeItem('fainaya_token');
}

export async function apiFetch(endpoint: string, options: RequestInit = {}) {
  const token = getToken();
  const headers = new Headers(options.headers || {});
  
  if (!headers.has('Content-Type') && !(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json');
  }

  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  const res = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });

  if (res.status === 401) {
    removeToken();
  }

  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error || 'Terjadi kesalahan sistem');
  }

  return data;
}
