import { useEffect, useMemo, useRef, useState } from 'preact/hooks';
import {
  cancelLogsWarmup,
  getLogSites,
  getLogStats,
  getLogsWarmupStatus,
  startLogsWarmup,
  type WarmupReport,
} from '../services/logsService';
import type { LogStats } from '../types/logs';
import LogsHitsChart from './LogsHitsChart';
import { getAuthToken } from '../utils/adminAuth';

function defaultRange() {
  const to = new Date();
  const from = new Date();
  from.setMonth(from.getMonth() - 1);
  return {
    from: from.toISOString().slice(0, 10),
    to: to.toISOString().slice(0, 10),
  };
}

export default function AdminLogsPage() {
  const initial = useMemo(defaultRange, []);
  const [from, setFrom] = useState(initial.from);
  const [to, setTo] = useState(initial.to);
  const [site, setSite] = useState('');
  const [sites, setSites] = useState<string[]>([]);
  const [stats, setStats] = useState<LogStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [adminToken, setAdminToken] = useState<string | null>(null);
  const [warmup, setWarmup] = useState<WarmupReport | null>(null);
  const [warmupError, setWarmupError] = useState<string | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    setAdminToken(getAuthToken());
  }, []);

  useEffect(() => {
    if (!adminToken) return;
    getLogsWarmupStatus(adminToken)
      .then(setWarmup)
      .catch(() => {
        /* status is optional */
      });
  }, [adminToken]);

  useEffect(() => {
    if (!adminToken || !warmup?.in_progress) {
      if (pollRef.current) {
        clearInterval(pollRef.current);
        pollRef.current = null;
      }
      return;
    }
    pollRef.current = setInterval(() => {
      getLogsWarmupStatus(adminToken)
        .then(setWarmup)
        .catch(() => {
          /* keep last state */
        });
    }, 2000);
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
      pollRef.current = null;
    };
  }, [adminToken, warmup?.in_progress]);

  async function onWarmup(force = false) {
    if (!adminToken) return;
    setWarmupError(null);
    try {
      const res = await startLogsWarmup(adminToken, force);
      setWarmup(res);
    } catch (err) {
      setWarmupError(err instanceof Error ? err.message : 'Error desconocido');
    }
  }

  async function onCancelWarmup() {
    if (!adminToken) return;
    try {
      const res = await cancelLogsWarmup(adminToken);
      setWarmup(res);
    } catch (err) {
      setWarmupError(err instanceof Error ? err.message : 'Error desconocido');
    }
  }

  useEffect(() => {
    getLogSites()
      .then((res) => {
        const list = res.sites ?? [];
        setSites(list);
        if (!site && list.length > 0) {
          setSite(list[0]);
        }
      })
      .catch(() => {
        /* site list is optional */
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const data = await getLogStats({ from, to, site: site || undefined, topN: 25 });
      setStats(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Error desconocido');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function onSubmit(e: Event) {
    e.preventDefault();
    load();
  }

  function exportCSV() {
    if (!stats) return;
    const lines = ['date,hits', ...stats.byDay.map((d) => `${d.date},${d.hits}`)];
    const blob = new Blob([lines.join('\n')], { type: 'text/csv;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `logs_${from || 'all'}_${to || 'all'}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <div class="space-y-6">
      {adminToken && (
        <div class="bg-[#1a1c22] rounded-lg p-4 flex flex-col md:flex-row md:items-center gap-3">
          <div class="flex-1 text-sm text-gray-300">
            <div class="font-semibold text-brand-blue mb-1">Caché de resúmenes</div>
            {warmup ? (
              warmup.in_progress ? (
                <span>
                  Procesando {warmup.built + warmup.cached + warmup.failed} / {warmup.total}{' '}
                  ficheros (construidos: {warmup.built}, en caché: {warmup.cached}
                  {warmup.failed > 0 ? `, fallidos: ${warmup.failed}` : ''})
                </span>
              ) : warmup.total > 0 ? (
                <span>
                  Última ejecución: {warmup.built} construidos, {warmup.cached} en caché
                  {warmup.failed > 0 ? `, ${warmup.failed} fallidos` : ''}
                  {warmup.cancelled ? ' (cancelada)' : ''}.
                </span>
              ) : (
                <span class="text-gray-500">Sin ejecutar todavía.</span>
              )
            ) : (
              <span class="text-gray-500">Cargando estado…</span>
            )}
            {warmupError && <div class="text-red-400 mt-1">{warmupError}</div>}
          </div>
          {warmup?.in_progress ? (
            <button
              type="button"
              onClick={onCancelWarmup}
              class="bg-gray-700 hover:bg-gray-600 text-white px-3 py-2 rounded"
            >
              Cancelar
            </button>
          ) : (
            <>
              <button
                type="button"
                onClick={() => onWarmup(false)}
                class="bg-brand-blue hover:bg-blue-600 text-white px-3 py-2 rounded"
              >
                Regenerar caché
              </button>
              <button
                type="button"
                onClick={() => onWarmup(true)}
                title="Reconstruye todos los resúmenes ignorando los existentes"
                class="bg-gray-700 hover:bg-gray-600 text-white px-3 py-2 rounded"
              >
                Forzar
              </button>
            </>
          )}
        </div>
      )}

      <form
        onSubmit={onSubmit}
        class="flex flex-col md:flex-row md:items-end gap-3 bg-[#1a1c22] p-4 rounded-lg"
      >
        <label class="flex flex-col text-sm text-gray-300">
          Desde
          <input
            type="date"
            value={from}
            onInput={(e) => setFrom((e.target as HTMLInputElement).value)}
            class="bg-[#0f1115] border border-gray-700 rounded px-2 py-1 text-white"
          />
        </label>
        <label class="flex flex-col text-sm text-gray-300">
          Hasta
          <input
            type="date"
            value={to}
            onInput={(e) => setTo((e.target as HTMLInputElement).value)}
            class="bg-[#0f1115] border border-gray-700 rounded px-2 py-1 text-white"
          />
        </label>
        <label class="flex flex-col text-sm text-gray-300">
          Web
          <select
            value={site}
            onChange={(e) => setSite((e.target as HTMLSelectElement).value)}
            class="bg-[#0f1115] border border-gray-700 rounded px-2 py-1 text-white min-w-[12rem]"
          >
            <option value="">Todas</option>
            {sites.map((s) => (
              <option value={s} key={s}>
                {s}
              </option>
            ))}
          </select>
        </label>
        <button
          type="submit"
          disabled={loading}
          class="bg-brand-pink hover:bg-pink-500 text-white px-4 py-2 rounded disabled:opacity-50"
        >
          {loading ? 'Cargando…' : 'Aplicar'}
        </button>
        <button
          type="button"
          onClick={exportCSV}
          disabled={!stats}
          class="bg-gray-700 hover:bg-gray-600 text-white px-4 py-2 rounded disabled:opacity-50"
        >
          Exportar CSV
        </button>
      </form>

      {error && (
        <div class="bg-red-900/40 border border-red-700 text-red-200 px-4 py-2 rounded">
          {error}
        </div>
      )}

      {stats && (
        <>
          <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
            <Summary label="Visitas totales" value={stats.totalHits.toLocaleString('es-ES')} />
            <Summary
              label="Humanos (estimado)"
              value={`${stats.humanHits.toLocaleString('es-ES')} (${pct(stats.humanHits, stats.totalHits)}%)`}
            />
            <Summary
              label="Bots (UA conocido)"
              value={`${stats.botHits.toLocaleString('es-ES')} (${pct(stats.botHits, stats.totalHits)}%)`}
            />
            <Summary
              label="Scanners (rutas)"
              value={`${stats.scannerHits.toLocaleString('es-ES')} (${pct(stats.scannerHits, stats.totalHits)}%)`}
            />
            <Summary label="Visitantes únicos" value={stats.uniqueIPs.toLocaleString('es-ES')} />
            <Summary label="URLs únicas" value={stats.topUrls.length.toLocaleString('es-ES')} />
            <Summary label="Días con tráfico" value={stats.byDay.length.toLocaleString('es-ES')} />
          </div>

          <LogsHitsChart data={stats.byDay} />

          <div class="grid md:grid-cols-2 gap-4">
            <Table
              title="Top URLs"
              headers={['URL', 'Visitas']}
              rows={stats.topUrls.map((u) => [u.url, u.hits.toLocaleString('es-ES')])}
            />
            <Table
              title="Top visitantes (anónimos)"
              headers={['Visitante', 'Visitas']}
              rows={stats.topIps.map((i) => [i.ip, i.hits.toLocaleString('es-ES')])}
            />
            <Table
              title="Códigos de respuesta"
              headers={['Status', 'Visitas']}
              rows={stats.byStatus.map((s) => [String(s.status), s.hits.toLocaleString('es-ES')])}
            />
            <Table
              title="Resumen mensual"
              headers={['Mes', 'Visitas']}
              rows={stats.byMonth.map((m) => [m.month, m.hits.toLocaleString('es-ES')])}
            />
          </div>

          <div>
            <h2 class="text-brand-pink font-semibold mb-3 mt-2">
              Bots y scanners
            </h2>
            <p class="text-sm text-gray-400 mb-3">
              Clasificación heurística: el User-Agent contra una lista de bots
              conocidos y la URL contra rutas típicas de escaneo de
              vulnerabilidades. Si esta sección sale vacía, pulsa <em>Forzar</em>
              en la caché de resúmenes para regenerar con la nueva versión.
            </p>
            <div class="grid md:grid-cols-2 gap-4">
              <Table
                title="Top User-Agents"
                headers={['User-Agent', 'Visitas']}
                rows={stats.topUA.map((u) => [u.userAgent, u.hits.toLocaleString('es-ES')])}
              />
              <Table
                title="Top User-Agents bot"
                headers={['User-Agent', 'Visitas']}
                rows={stats.topBotUA.map((u) => [u.userAgent, u.hits.toLocaleString('es-ES')])}
              />
              <Table
                title="Top rutas de scanner"
                headers={['Ruta', 'Visitas']}
                rows={stats.topScannerPath.map((u) => [u.url, u.hits.toLocaleString('es-ES')])}
              />
            </div>
          </div>
        </>
      )}
    </div>
  );
}

function pct(part: number, total: number) {
  if (!total) return '0';
  return ((part / total) * 100).toFixed(1);
}

function Summary({ label, value }: { label: string; value: string }) {
  return (
    <div class="bg-[#1a1c22] rounded-lg p-4">
      <div class="text-xs text-gray-400 uppercase tracking-wide">{label}</div>
      <div class="text-2xl font-semibold text-white mt-1">{value}</div>
    </div>
  );
}

function Table({
  title,
  headers,
  rows,
}: {
  title: string;
  headers: string[];
  rows: string[][];
}) {
  return (
    <div class="bg-[#1a1c22] rounded-lg p-4">
      <h3 class="text-brand-blue font-semibold mb-3">{title}</h3>
      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left text-gray-300">
          <thead class="text-xs uppercase text-gray-400 border-b border-gray-700">
            <tr>
              {headers.map((h) => (
                <th class="py-2 pr-3" key={h}>
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 && (
              <tr>
                <td class="py-3 text-gray-500" colSpan={headers.length}>
                  Sin datos
                </td>
              </tr>
            )}
            {rows.map((row, idx) => (
              <tr key={idx} class="border-b border-gray-800/60">
                {row.map((cell, i) => (
                  <td class="py-2 pr-3 break-all" key={i}>
                    {cell}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
