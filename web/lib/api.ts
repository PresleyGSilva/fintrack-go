const API_URL =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8084";

const PREFIX = "/api/v1";

function getHeaders(extra?: HeadersInit): Headers {
  const h = new Headers(extra);
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("fintrack_token");
    if (token) h.set("Authorization", `Bearer ${token}`);
    const orgId = localStorage.getItem("fintrack_org_id");
    const user = localStorage.getItem("fintrack_user");
    if (orgId) h.set("X-Org-ID", orgId);
    if (user) h.set("X-User", user);
  }
  if (!h.has("Content-Type")) h.set("Content-Type", "application/json");
  return h;
}

/** Unwraps the `{ data: ... }` envelope, falling back to the raw body. */
function unwrap<T>(body: unknown): T {
  if (body !== null && typeof body === "object" && "data" in body) {
    return body.data as T;
  }
  return body as T;
}

function errorMessage(fallback: string, text: string): Error {
  if (text.startsWith("{")) {
    try {
      const parsed: unknown = JSON.parse(text);
      if (
        parsed !== null &&
        typeof parsed === "object" &&
        "error" in parsed &&
        parsed.error !== null &&
        typeof parsed.error === "object" &&
        "message" in parsed.error &&
        typeof parsed.error.message === "string"
      ) {
        return new Error(parsed.error.message);
      }
    } catch {
      // fall through to raw text
    }
  }
  return new Error(text || fallback);
}

async function refreshToken(): Promise<boolean> {
  try {
    const res = await fetch(`${API_URL}${PREFIX}/auth/refresh`, {
      method: "POST",
      credentials: "include",
    });
    if (!res.ok) return false;
    const body: unknown = await res.json().catch(() => null);
    const data = unwrap<{ access_token?: string }>(body);
    const token = data?.access_token ?? null;
    if (token && typeof window !== "undefined") {
      localStorage.setItem("fintrack_token", token);
      return true;
    }
    return res.ok;
  } catch {
    return false;
  }
}

export async function api<T = unknown>(
  path: string,
  init: RequestInit = {},
  retried = false,
): Promise<T> {
  const res = await fetch(`${API_URL}${PREFIX}${path}`, {
    ...init,
    headers: getHeaders(init.headers),
    credentials: "include",
  });
  if (res.status === 401 && !retried) {
    const ok = await refreshToken();
    if (ok) return api<T>(path, init, true);
    if (typeof window !== "undefined") window.location.href = "/login";
    throw new Error("Não autenticado");
  }
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw errorMessage(`Erro ${res.status}`, text);
  }
  if (res.status === 204) return null as T;
  const body: unknown = await res.json().catch(() => null);
  return unwrap<T>(body);
}

export interface Org {
  id: string;
  name?: string;
}

export async function loginRequest(
  email: string,
  password: string,
): Promise<{ access_token: string; orgs: Org[] }> {
  const res = await fetch(`${API_URL}${PREFIX}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw errorMessage("Falha no login", text);
  }
  const body: unknown = await res.json();
  const payload = unwrap<{ access_token?: string; orgs?: Org[] }>(body);
  const access_token = payload?.access_token ?? "";
  const orgs: Org[] = payload?.orgs ?? [];
  if (typeof window !== "undefined") {
    localStorage.setItem("fintrack_token", access_token);
    localStorage.setItem("fintrack_orgs", JSON.stringify(orgs));
    // REGRA DURA: org ativa é sempre orgs[0].id
    if (orgs.length > 0) localStorage.setItem("fintrack_org_id", orgs[0].id);
    localStorage.setItem("fintrack_user", email);
  }
  return { access_token, orgs };
}

export function logout(): void {
  if (typeof window !== "undefined") {
    localStorage.removeItem("fintrack_token");
    localStorage.removeItem("fintrack_orgs");
    localStorage.removeItem("fintrack_org_id");
    fetch(`${API_URL}${PREFIX}/auth/logout`, {
      method: "POST",
      credentials: "include",
      headers: getHeaders(),
    }).catch(() => undefined);
    window.location.href = "/login";
  }
}

export { API_URL, PREFIX };
