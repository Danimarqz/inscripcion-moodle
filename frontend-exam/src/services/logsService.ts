import { request } from './api';
import type { LogStats, LogStatsQuery } from '../types/logs';

export async function getLogStats(query: LogStatsQuery): Promise<LogStats> {
  const params = new URLSearchParams();
  if (query.from) params.set('from', query.from);
  if (query.to) params.set('to', query.to);
  if (query.site) params.set('site', query.site);
  if (query.topN) params.set('topN', String(query.topN));
  const qs = params.toString();
  return request<LogStats>(`/logs/stats${qs ? `?${qs}` : ''}`, {
    method: 'GET',
    timeoutMs: 240_000,
  });
}

export async function getLogSites(): Promise<{ sites: string[] }> {
  return request<{ sites: string[] }>('/logs/sites', { method: 'GET' });
}

export interface WarmupReport {
  total: number;
  built: number;
  cached: number;
  failed: number;
  errors?: string[];
  started_at: string;
  ended_at: string;
  duration: number;
  in_progress: boolean;
  cancelled?: boolean;
}

export async function startLogsWarmup(token: string, force = false): Promise<WarmupReport> {
  return request<WarmupReport>(`/admin/logs/warmup${force ? '?force=true' : ''}`, {
    method: 'POST',
    token,
  });
}

export async function getLogsWarmupStatus(token: string): Promise<WarmupReport> {
  return request<WarmupReport>('/admin/logs/warmup/status', {
    method: 'GET',
    token,
  });
}

export async function cancelLogsWarmup(token: string): Promise<WarmupReport> {
  return request<WarmupReport>('/admin/logs/warmup/cancel', {
    method: 'POST',
    token,
  });
}
