import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { request, ApiError } from '../api';

beforeEach(() => {
  vi.stubGlobal('fetch', vi.fn());
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('request', () => {
  it('returns JSON on success', async () => {
    (globalThis.fetch as any).mockResolvedValue({
      ok: true, status: 200, json: () => Promise.resolve({ id: 1 }),
      text: () => Promise.resolve(''),
    });
    const result = await request<any>('/api/test');
    expect(result).toEqual({ id: 1 });
  });

  it('includes credentials for admin', async () => {
    (globalThis.fetch as any).mockResolvedValue({
      ok: true, status: 200, json: () => Promise.resolve({}),
      text: () => Promise.resolve(''),
    });
    await request<any>('/admin/users');
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({ credentials: 'include' }),
    );
  });

  it('throws ApiError on 404', async () => {
    (globalThis.fetch as any).mockResolvedValue({
      ok: false, status: 404, statusText: 'Not Found',
      text: () => Promise.resolve(''),
    });
    await expect(request<any>('/api/missing')).rejects.toThrow(ApiError);
  });

  it('parses error detail', async () => {
    (globalThis.fetch as any).mockResolvedValue({
      ok: false, status: 400, statusText: 'Bad Request',
      text: () => Promise.resolve(JSON.stringify({ detail: 'Campo requerido' })),
    });
    await expect(request<any>('/api/bad')).rejects.toThrow('Campo requerido');
  });

  it('throws timeout on abort', async () => {
    const err = new DOMException('aborted', 'AbortError');
    (globalThis.fetch as any).mockRejectedValue(err);
    await expect(request<any>('/api/test')).rejects.toThrow('La solicitud tard');
  });

  it('sends Bearer only for JWT tokens', async () => {
    (globalThis.fetch as any).mockResolvedValue({
      ok: true, status: 200, json: () => Promise.resolve({}),
      text: () => Promise.resolve(''),
    });
    await request<any>('/api/test', { token: 'cookie-session' });
    const call1 = (globalThis.fetch as any).mock.calls[0][1];
    expect(call1.headers.Authorization).toBeUndefined();

    await request<any>('/api/test', { token: 'eyJhbGci.eyJzdWIiOiIx' });
    const call2 = (globalThis.fetch as any).mock.calls[1][1];
    expect(call2.headers.Authorization).toContain('Bearer');
  });
});
