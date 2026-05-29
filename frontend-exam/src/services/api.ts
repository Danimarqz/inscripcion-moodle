const API_URL = import.meta.env.PUBLIC_API_URL;

const TIMEOUT_MS = 15_000;

interface RequestOptions extends RequestInit {
  token?: string;
  timeoutMs?: number;
}

export class ApiError extends Error {
  constructor(public message: string, public status?: number) {
    super(message);
    this.name = 'ApiError';
  }
}

export async function request<T>(endpoint: string, options: RequestOptions = {}): Promise<T> {
  // Discard any caller-supplied signal; we manage our own timeout.
  const { token, headers, signal: _ignored, timeoutMs, ...rest } = options;

  const controller = new AbortController();
  const timeoutId = setTimeout(
    () => controller.abort(new DOMException('Request timed out', 'TimeoutError')),
    timeoutMs ?? TIMEOUT_MS,
  );

  const defaultHeaders: HeadersInit = {
    'Content-Type': 'application/json',
  };
  // Admin auth now rides the httpOnly cookie (sent via credentials:'include').
  // Only attach a Bearer header when given a real JWT — the in-memory session
  // marker (a non-JWT placeholder used after a reload) must not be sent.
  if (token && token.includes('.')) {
    defaultHeaders['Authorization'] = `Bearer ${token}`;
  }

  try {
    const response = await fetch(`${API_URL}${endpoint}`, {
      headers: { ...defaultHeaders, ...headers },
      // Send the auth cookie only for admin endpoints. Public/student requests
      // stay uncredentialed so they keep working under a wildcard CORS origin
      // (a credentialed request requires a specific Access-Control-Allow-Origin).
      credentials: endpoint.startsWith('/admin') ? 'include' : 'same-origin',
      signal: controller.signal,
      ...rest,
    });

    if (!response.ok) {
      // Try to parse as JSON first ({"detail": "..."}), fall back to plain text.
      const text = await response.text().catch(() => '');
      let errorMessage = `Error ${response.status}: ${response.statusText}`;
      try {
        const errorData = JSON.parse(text) as Record<string, unknown>;
        if (typeof errorData.detail === 'string') errorMessage = errorData.detail;
      } catch {
        if (text.trim()) errorMessage = text.trim();
      }
      throw new ApiError(errorMessage, response.status);
    }

    if (response.status === 204) {
      return {} as T;
    }

    return response.json() as Promise<T>;
  } catch (err) {
    if (err instanceof DOMException && (err.name === 'AbortError' || err.name === 'TimeoutError')) {
      throw new ApiError('La solicitud tardó demasiado. Por favor, inténtalo de nuevo.', 408);
    }
    throw err;
  } finally {
    clearTimeout(timeoutId);
  }
}
