import { useEffect, useMemo, useState } from 'preact/hooks';
import type { Exam, ExamOfficialResult, ImportOfficialResultsSummary } from '../types/exam';
import { getOfficialResults } from '../services/adminService';
import { useAsyncTask } from '../hooks/useAsyncTask';
import OfficialResultImport from './official-results/OfficialResultImport';
import OfficialResultForm from './official-results/OfficialResultForm';
import OfficialResultsTable from './official-results/OfficialResultsTable';

interface OfficialResultsManagerProps {
  exams: Exam[];
  token: string;
}

export default function OfficialResultsManager({ exams, token }: OfficialResultsManagerProps) {
  const [selectedExamId, setSelectedExamId] = useState<string>('');
  const [results, setResults] = useState<ExamOfficialResult[]>([]);
  const [totalResults, setTotalResults] = useState<number>(0);
  const {
    loading: resultsLoading,
    error: resultsError,
    run,
    setError: setResultsError,
  } = useAsyncTask();
  const [feedback, setFeedback] = useState<string | null>(null);
  const [summary, setSummary] = useState<ImportOfficialResultsSummary | null>(null);

  const [sortBy, setSortBy] = useState<'dni' | 'nombre' | 'apellidos' | 'usuario' | 'creado'>(
    'apellidos',
  );
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');
  const [pageSize, setPageSize] = useState<number>(100);
  const [currentPage, setCurrentPage] = useState<number>(1);

  const selectedExamName = useMemo(() => {
    const numericId = Number(selectedExamId);
    return exams.find((exam) => exam.id === numericId)?.name ?? '';
  }, [exams, selectedExamId]);

  useEffect(() => {
    setCurrentPage(1);
    setSummary(null);
    setTotalResults(0);
  }, [selectedExamId]);

  useEffect(() => {
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
      setResultsError('Identificador de examen inválido.');
      return;
    }

    async function load(page: number) {
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

    void load(currentPage).catch(() => {
      setResults([]);
      setTotalResults(0);
    });
  }, [run, selectedExamId, token, sortBy, sortDirection, pageSize, currentPage]);

  function handleSelectExam(event: Event) {
    const value = (event.target as HTMLSelectElement).value;
    setSelectedExamId(value);
  }

  async function refreshResults() {
    const examNumericId = Number(selectedExamId);
    if (Number.isNaN(examNumericId)) return;

    try {
      const refreshed = await getOfficialResults(examNumericId, token, {
        limit: pageSize,
        offset: 0,
        orderBy: sortBy,
        orderDir: sortDirection,
      });
      setResults(refreshed.results);
      setTotalResults(refreshed.total);
      setCurrentPage(1);
    } catch (err) {
      setResultsError(err instanceof Error ? err.message : String(err));
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

  const handleImportSuccess = async (importSummary: ImportOfficialResultsSummary) => {
    setSummary(importSummary);
    setFeedback(
      `Importación completada: ${importSummary.imported_results}/${importSummary.total_rows} filas registradas.`,
    );
    await refreshResults();
  };

  const handleCreateSuccess = async (message: string) => {
    setFeedback(message);
    setSummary(null);
    await refreshResults();
  };

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
        <p className="text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md mb-4">
          {resultsError}
        </p>
      )}
      {feedback && (
        <p className="text-green-400 bg-green-400/10 border border-green-500 p-4 rounded-md mb-4">
          {feedback}
        </p>
      )}

      {summary && (
        <p className="text-xs text-brand-blue mb-4">
          Resumen: {summary.imported_results}/{summary.total_rows} filas guardadas, usuarios
          actualizados: {summary.updated_users}.
        </p>
      )}

      {selectedExamId && (
        <OfficialResultForm
          examId={selectedExamId}
          token={token}
          onSuccess={handleCreateSuccess}
          onError={(msg) => {
            setFeedback(null);
            setResultsError(msg);
          }}
        />
      )}

      <OfficialResultImport
        examId={selectedExamId}
        token={token}
        onSuccess={handleImportSuccess}
        onError={(msg) => {
          setSummary(null);
          setFeedback(null);
          setResultsError(msg);
        }}
      />

      {!selectedExamId && (
        <p className="text-sm text-brand-blue">
          Selecciona un examen para ver los resultados oficiales.
        </p>
      )}

      {selectedExamId && resultsLoading && (
        <p className="text-brand-yellow">Cargando resultados oficiales...</p>
      )}

      {selectedExamId && !resultsLoading && results.length === 0 && !resultsError && (
        <p className="text-sm text-brand-pink">
          No hay resultados oficiales importados para este examen.
        </p>
      )}

      {selectedExamId && !resultsLoading && results.length > 0 && (
        <OfficialResultsTable
          selectedExamName={selectedExamName}
          results={results}
          totalResults={totalResults}
          pageSize={pageSize}
          currentPage={currentPage}
          sortBy={sortBy}
          sortDirection={sortDirection}
          onPageSizeChange={(size) => {
            setPageSize(size);
            setCurrentPage(1);
          }}
          onPageChange={setCurrentPage}
          onSort={handleSort}
        />
      )}
    </section>
  );
}


