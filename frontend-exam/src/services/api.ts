const API_URL = import.meta.env.PUBLIC_API_URL;

const TIMEOUT_MS = 15_000;

interface RequestOptions extends RequestInit {
  token?: string;
}

export class ApiError extends Error {
  constructor(public message: string, public status?: number) {
    super(message);
    this.name = 'ApiError';
  }
}

export async function request<T>(endpoint: string, options: RequestOptions = {}): Promise<T> {
  const { token, headers, signal: callerSignal, ...rest } = options;

  const timeoutSignal = AbortSignal.timeout(TIMEOUT_MS);
  const combinedSignal = callerSignal
    ? AbortSignal.any([callerSignal, timeoutSignal])
    : timeoutSignal;

  const defaultHeaders: HeadersInit = {
    'Content-Type': 'application/json',
  };
  if (token) {
    defaultHeaders['Authorization'] = `Bearer ${token}`;
  }

  try {
    const response = await fetch(`${API_URL}${endpoint}`, {
      headers: { ...defaultHeaders, ...headers },
      signal: combinedSignal,
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
  }
}
