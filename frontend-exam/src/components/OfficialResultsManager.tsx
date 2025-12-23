import { useEffect, useMemo, useRef, useState } from 'preact/hooks';

import type {
  Exam,
  ExamOfficialResult,
  ImportOfficialResultsSummary,
  SortableHeaderProps,
  CreateOfficialResultPayload,
} from '../types/exam';
import {
  createOfficialResult,
  getOfficialResults,
  importOfficialResults,
} from '../services/adminService';
import { useAsyncTask } from '../hooks/useAsyncTask';

interface OfficialResultsManagerProps {
  exams: Exam[];
  token: string;
}

export default function OfficialResultsManager({ exams, token }: OfficialResultsManagerProps) {
  const [selectedExamId, setSelectedExamId] = useState<string>('');
  const [results, setResults] = useState<ExamOfficialResult[]>([]);
  const [totalResults, setTotalResults] = useState<number>(0);
  const { loading: resultsLoading, error: resultsError, run, setError: setResultsError } = useAsyncTask();
  const [feedback, setFeedback] = useState<string | null>(null);
  const [importing, setImporting] = useState<boolean>(false);
  const [summary, setSummary] = useState<ImportOfficialResultsSummary | null>(null);
  const [manualResult, setManualResult] = useState<{ dni: string; apellido1: string; apellido2: string; nombre: string }>({
    dni: '',
    apellido1: '',
    apellido2: '',
    nombre: '',
  });
  const [creating, setCreating] = useState<boolean>(false);
  const [sortBy, setSortBy] = useState<'dni' | 'nombre' | 'apellidos' | 'usuario' | 'creado'>('apellidos');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');
  const [pageSize, setPageSize] = useState<number>(100);
  const [currentPage, setCurrentPage] = useState<number>(1);

  const fileInputRef = useRef<HTMLInputElement>(null);

  const selectedExamName = useMemo(() => {
    const numericId = Number(selectedExamId);
    return exams.find((exam) => exam.id === numericId)?.name ?? '';
  }, [exams, selectedExamId]);

  useEffect(() => {
    setCurrentPage(1);
    setManualResult({ dni: '', apellido1: '', apellido2: '', nombre: '' });
    setSummary(null);
    setTotalResults(0);
  }, [selectedExamId]);

  useEffect(() => {
    async function load(examNumericId: number, page: number) {
      await run(async () => {
        setFeedback(null);
        const offset = Math.max(0, (page - 1) * pageSize);
        const data = await getOfficialResults(examNumericId, token, {
          limit: pageSize,
          offset,
          orderBy: sortBy,
          orderDir: sortDirection,
        });
        setResults(data.results);
        setTotalResults(data.total);
        const totalPages = Math.max(1, Math.ceil(Math.max(data.total, 1) / pageSize));
        if (page > totalPages) {
          setCurrentPage(totalPages);
        }
      });
    }

    if (!selectedExamId) {
      setResults([]);
      setTotalResults(0);
      setSummary(null);
      setFeedback(null);
      setResultsError(null);
      return;
    }

    const examNumericId = Number(selectedExamId);
    if (Number.isNaN(examNumericId)) {
      setResults([]);
      setTotalResults(0);
      setResultsError('Identificador de examen invalido.');
      return;
    }

    void load(examNumericId, currentPage).catch(() => {
      setResults([]);
      setTotalResults(0);
    });
  }, [run, selectedExamId, token, sortBy, sortDirection, pageSize, currentPage]);

  function handleSelectExam(event: Event) {
    const value = (event.target as HTMLSelectElement).value;
    setSelectedExamId(value);
  }

  async function handleImport(event: Event) {
    event.preventDefault();
    if (!selectedExamId) {
      setResultsError('Selecciona un examen antes de importar.');
      return;
    }

    const examNumericId = Number(selectedExamId);
    if (Number.isNaN(examNumericId)) {
      setResultsError('Identificador de examen invalido.');
      return;
    }

    const file = fileInputRef.current?.files?.[0];
    if (!file) {
      setResultsError('Selecciona un archivo Excel para importar.');
      return;
    }

    setImporting(true);
    setResultsError(null);

    try {
      const importSummary = await importOfficialResults(examNumericId, file, token);
      setSummary(importSummary);
      setFeedback(
        `Importacion completada: ${importSummary.imported_results}/${importSummary.total_rows} filas registradas.`,
      );
      const refreshed = await run(
        () =>
          getOfficialResults(examNumericId, token, {
            limit: pageSize,
            offset: 0,
            orderBy: sortBy,
            orderDir: sortDirection,
          }),
        { suppressLoading: true },
      );
      setResults(refreshed.results);
      setTotalResults(refreshed.total);
      setCurrentPage(1);
    } catch (err) {
      setSummary(null);
      setFeedback(null);
      setResultsError(err instanceof Error ? err.message : String(err));
    } finally {
      setImporting(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  }

  async function handleCreateResult(event: Event) {
    event.preventDefault();
    if (!selectedExamId) {
      setResultsError('Selecciona un examen antes de anadir un registro.');
      return;
    }

    const examNumericId = Number(selectedExamId);
    if (Number.isNaN(examNumericId)) {
      setResultsError('Identificador de examen invalido.');
      return;
    }

    const dni = manualResult.dni.trim();
    const apellido1 = manualResult.apellido1.trim();
    const apellido2 = manualResult.apellido2.trim();
    const nombre = manualResult.nombre.trim();

    if (!dni || !apellido1 || !nombre) {
      setResultsError('DNI, apellido 1 y nombre son obligatorios.');
      return;
    }

    const payload: CreateOfficialResultPayload = {
      dni,
      apellido_1: apellido1,
      nombre,
      ...(apellido2 ? { apellido_2: apellido2 } : {}),
    };

    setCreating(true);
    setResultsError(null);
    setFeedback(null);

    try {
      await createOfficialResult(examNumericId, payload, token);
      setSummary(null);
      setFeedback('Registro agregado correctamente.');
      setManualResult({ dni: '', apellido1: '', apellido2: '', nombre: '' });
      setCurrentPage(1);
      const refreshed = await getOfficialResults(examNumericId, token, {
        limit: pageSize,
        offset: 0,
        orderBy: sortBy,
        orderDir: sortDirection,
      });
      setResults(refreshed.results);
      setTotalResults(refreshed.total);
    } catch (err) {
      setFeedback(null);
      setResultsError(err instanceof Error ? err.message : String(err));
    } finally {
      setCreating(false);
    }
  }

  const handleSort = (key: 'dni' | 'nombre' | 'apellidos' | 'usuario' | 'creado') => {
    if (sortBy === key) {
      setSortDirection((prev) => (prev === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortBy(key);
      setSortDirection('asc');
    }
    setCurrentPage(1);
  };

  const totalPages = useMemo(
    () => Math.max(1, Math.ceil(Math.max(totalResults, 1) / pageSize)),
    [totalResults, pageSize],
  );

  useEffect(() => {
    if (currentPage > totalPages) {
      setCurrentPage(totalPages);
    }
  }, [currentPage, totalPages]);

  return (
    <section className="mt-8 space-y-6">
      <label className="block mb-4">
        <span className="block font-semibold mb-2 text-brand-pink tracking-[0.35em] uppercase text-xs">
          Selecciona un examen
        </span>
        <select
          value={selectedExamId}
          onChange={handleSelectExam}
          className="w-full max-w-sm px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white transition-colors focus:outline-none focus:ring-2 focus:ring-brand-blue focus:border-brand-blue"
        >
          <option value="">-- Escoge un examen --</option>
          {exams.map((exam) => (
            <option key={exam.id} value={exam.id}>
              {exam.name} (ID: {exam.id})
            </option>
          ))}
        </select>
      </label>

      {resultsError && (
        <p className="text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md mb-4">{resultsError}</p>
      )}
      {feedback && (
        <p className="text-green-400 bg-green-400/10 border border-green-500 p-4 rounded-md mb-4">{feedback}</p>
      )}

      {summary && (
        <p className="text-xs text-brand-blue mb-4">
          Resumen: {summary.imported_results}/{summary.total_rows} filas guardadas, usuarios actualizados: {summary.updated_users}.
        </p>
      )}

      {selectedExamId && (
        <form
          className="grid gap-3 md:grid-cols-[repeat(5,minmax(0,1fr))] items-end bg-[#14161d] border border-brand-blue-soft rounded-2xl p-4 shadow-xl"
          onSubmit={handleCreateResult}
        >
          <div className="flex flex-col gap-1">
            <label className="text-xs text-brand-pink font-semibold uppercase tracking-[0.25em]">DNI</label>
            <input
              type="text"
              value={manualResult.dni}
              onInput={(event) =>
                setManualResult((prev) => ({ ...prev, dni: (event.target as HTMLInputElement).value }))
              }
              className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue"
              placeholder="00000000A"
              disabled={!selectedExamId || creating}
            />
          </div>
          <div className="flex flex-col gap-1">
            <label className="text-xs text-brand-pink font-semibold uppercase tracking-[0.25em]">Apellido 1</label>
            <input
              type="text"
              value={manualResult.apellido1}
              onInput={(event) =>
                setManualResult((prev) => ({ ...prev, apellido1: (event.target as HTMLInputElement).value }))
              }
              className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue"
              placeholder="Primer apellido"
              disabled={!selectedExamId || creating}
            />
          </div>
          <div className="flex flex-col gap-1">
            <label className="text-xs text-brand-pink font-semibold uppercase tracking-[0.25em]">Apellido 2 (opcional)</label>
            <input
              type="text"
              value={manualResult.apellido2}
              onInput={(event) =>
                setManualResult((prev) => ({ ...prev, apellido2: (event.target as HTMLInputElement).value }))
              }
              className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue"
              placeholder="Segundo apellido"
              disabled={!selectedExamId || creating}
            />
          </div>
          <div className="flex flex-col gap-1">
            <label className="text-xs text-brand-pink font-semibold uppercase tracking-[0.25em]">Nombre</label>
            <input
              type="text"
              value={manualResult.nombre}
              onInput={(event) =>
                setManualResult((prev) => ({ ...prev, nombre: (event.target as HTMLInputElement).value }))
              }
              className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue"
              placeholder="Nombre"
              disabled={!selectedExamId || creating}
            />
          </div>
          <div className="flex md:justify-end">
            <button
              type="submit"
              className="w-full md:w-auto py-2 px-5 rounded font-semibold bg-brand-pink text-white shadow-[0_10px_30px_rgba(237,87,150,0.25)] hover:bg-[#f3529f] transition-colors cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed"
              disabled={!selectedExamId || creating}
            >
              {creating ? 'Anadiendo...' : 'Anadir registro'}
            </button>
          </div>
        </form>
      )}

      <form className="flex flex-col gap-3 md:flex-row md:items-center mb-6" onSubmit={handleImport}>
        <label className="flex flex-col gap-1 text-sm">
          <span className="text-brand-pink font-semibold">Importar Excel de resultados</span>
          <input
            ref={fileInputRef}
            type="file"
            accept=".xlsx, .xls, application/vnd.openxmlformats-officedocument.spreadsheetml.sheet, application/vnd.ms-excel"
            className="px-3 py-2 rounded border border-dashed border-brand-pink-soft bg-[#1f2229] text-white cursor-pointer focus:outline-none focus:ring-2 focus:ring-brand-pink"
            disabled={!selectedExamId || importing}
          />
        </label>
        <button
          type="submit"
          className="py-2 px-5 rounded mt-5 md:mt-6 md:ml-5 font-semibold bg-brand-blue text-white shadow-[0_10px_30px_rgba(15,153,188,0.25)] hover:bg-[#12b2d4] transition-colors cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed"
          disabled={!selectedExamId || importing}
        >
          {importing ? 'Importando...' : 'Importar Excel'}
        </button>
      </form>

      {!selectedExamId && (
        <p className="text-sm text-brand-blue">Selecciona un examen para ver los resultados oficiales.</p>
      )}

      {selectedExamId && resultsLoading && <p className="text-brand-yellow">Cargando resultados oficiales...</p>}

      {selectedExamId && !resultsLoading && results.length === 0 && !resultsError && (
        <p className="text-sm text-brand-pink">No hay resultados oficiales importados para este examen.</p>
      )}

      {selectedExamId && !resultsLoading && results.length > 0 && (
        <div className="bg-[#14161d] border border-brand-blue-soft rounded-2xl p-5 shadow-xl">
          <h3 className="text-lg font-semibold text-brand-pink mb-3">
            Listado oficial ({selectedExamName})
          </h3>
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between mb-4">
            <div className="text-sm text-brand-blue font-semibold">
              Total registros: {totalResults}
            </div>
            <div className="flex items-center gap-3">
              <label className="text-xs text-brand-yellow flex items-center gap-2">
                Filas por pagina
                <select
                  value={pageSize}
                  onChange={(event) => {
                    setPageSize(Number((event.target as HTMLSelectElement).value));
                    setCurrentPage(1);
                  }}
                  className="px-2 py-1 rounded border border-brand-yellow-soft bg-[#1f2229] text-white text-xs focus:outline-none focus:ring-2 focus:ring-brand-yellow"
                >
                  {[50, 100, 200, 500].map((size) => (
                    <option key={size} value={size}>
                      {size}
                    </option>
                  ))}
                </select>
              </label>
            </div>
          </div>
          <div className="overflow-x-auto">
            <table className="min-w-full text-sm text-left border border-brand-blue-soft rounded-xl overflow-hidden">
              <thead className="bg-[#1f2229] text-brand-yellow uppercase text-xs tracking-[0.3em]">
                <tr>
                  <SortableHeader
                    label="DNI Excel"
                    sortKey="dni"
                    activeKey={sortBy}
                    direction={sortDirection}
                    onSort={handleSort}
                  />
                  <SortableHeader
                    label="Nombre"
                    sortKey="nombre"
                    activeKey={sortBy}
                    direction={sortDirection}
                    onSort={handleSort}
                  />
                  <SortableHeader
                    label="Apellidos"
                    sortKey="apellidos"
                    activeKey={sortBy}
                    direction={sortDirection}
                    onSort={handleSort}
                  />
                  <SortableHeader
                    label="Usuario asociado"
                    sortKey="usuario"
                    activeKey={sortBy}
                    direction={sortDirection}
                    onSort={handleSort}
                  />
                  <SortableHeader
                    label="Creado"
                    sortKey="creado"
                    activeKey={sortBy}
                    direction={sortDirection}
                    onSort={handleSort}
                  />
                </tr>
              </thead>
              <tbody>
                {results.map((result) => {
                  const surname = [result.apellido_1, result.apellido_2].filter(Boolean).join(' ');
                  const user = result.user;
                  const associatedUser = user
                    ? `${user.name} ${user.surname} (${user.dni}${user.email ? ` - ${user.email}` : ''})`
                    : 'Sin usuario enlazado';

                  return (
                    <tr key={result.id} className="odd:bg-[#1b1e25] border-b border-brand-blue-soft last:border-b-0">
                      <td className="px-3 py-2 font-mono text-xs text-brand-blue">{result.dni_masked}</td>
                      <td className="px-3 py-2 text-white font-semibold">{result.nombre}</td>
                      <td className="px-3 py-2 text-white">{surname}</td>
                      <td className="px-3 py-2 text-brand-pink">{associatedUser}</td>
                      <td className="px-3 py-2 text-xs text-brand-yellow">
                        {new Date(result.created_at).toLocaleString()}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
          <PaginationControls
            totalItems={totalResults}
            pageSize={pageSize}
            currentPage={currentPage}
            onPageChange={setCurrentPage}
          />
        </div>
      )}
    </section>
  );
}

function SortableHeader({ label, sortKey, activeKey, direction, onSort }: SortableHeaderProps) {
  const isActive = activeKey === sortKey;
  const arrow = isActive ? (direction === 'asc' ? '↑' : '↓') : '↕';

  function handleClick() {
    onSort(sortKey);
  }

  return (
    <th className="px-3 py-2">
      <button
        type="button"
        onClick={handleClick}
        className={`flex items-center gap-1 text-xs font-semibold tracking-wide transition-colors ${
          isActive ? 'text-brand-pink' : 'text-gray-300 hover:text-brand-yellow'
        }`}
      >
        <span>{label}</span>
        <span className="text-xs">{arrow}</span>
      </button>
    </th>
  );
}

interface PaginationControlsProps {
  totalItems: number;
  pageSize: number;
  currentPage: number;
  onPageChange: (page: number) => void;
}

function PaginationControls({ totalItems, pageSize, currentPage, onPageChange }: PaginationControlsProps) {
  const totalPages = Math.max(1, Math.ceil(totalItems / pageSize));
  const startItem = Math.min(totalItems, (currentPage - 1) * pageSize + 1);
  const endItem = Math.min(totalItems, currentPage * pageSize);

  if (totalItems <= pageSize) {
    return null;
  }

  return (
    <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between mt-4 text-xs text-brand-blue">
      <div className="font-semibold">
        Mostrando {startItem}-{endItem} de {totalItems} registros - Página {currentPage} de {totalPages}
      </div>
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={() => onPageChange(Math.max(1, currentPage - 1))}
          className="px-3 py-1 rounded border border-brand-blue-soft text-brand-blue hover:bg-brand-blue-soft disabled:opacity-50 disabled:cursor-not-allowed"
          disabled={currentPage <= 1}
        >
          Anterior
        </button>
        <button
          type="button"
          onClick={() => onPageChange(Math.min(totalPages, currentPage + 1))}
          className="px-3 py-1 rounded border border-brand-blue-soft text-brand-blue hover:bg-brand-blue-soft disabled:opacity-50 disabled:cursor-not-allowed"
          disabled={currentPage >= totalPages}
        >
          Siguiente
        </button>
      </div>
    </div>
  );
}
