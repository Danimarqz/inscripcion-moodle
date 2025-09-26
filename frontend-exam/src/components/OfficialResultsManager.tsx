import { useEffect, useMemo, useRef, useState } from 'preact/hooks';

import type {
  Exam,
  ExamOfficialResult,
  ImportOfficialResultsSummary,
} from '../types/exam';
import {
  getOfficialResults,
  importOfficialResults,
} from '../services/adminService';

interface OfficialResultsManagerProps {
  exams: Exam[];
  token: string;
}

export default function OfficialResultsManager({ exams, token }: OfficialResultsManagerProps) {
  const [selectedExamId, setSelectedExamId] = useState<string>('');
  const [results, setResults] = useState<ExamOfficialResult[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [importing, setImporting] = useState<boolean>(false);
  const [summary, setSummary] = useState<ImportOfficialResultsSummary | null>(null);

  const fileInputRef = useRef<HTMLInputElement>(null);

  const selectedExamName = useMemo(() => {
    const numericId = Number(selectedExamId);
    return exams.find((exam) => exam.id === numericId)?.name ?? '';
  }, [exams, selectedExamId]);

  useEffect(() => {
    async function load(examNumericId: number) {
      setLoading(true);
      setError(null);
      setFeedback(null);
      try {
        const data = await getOfficialResults(examNumericId, token);
        setResults(data);
      } catch (err) {
        setResults([]);
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        setLoading(false);
      }
    }

    if (!selectedExamId) {
      setResults([]);
      setSummary(null);
      setFeedback(null);
      setError(null);
      return;
    }

    const examNumericId = Number(selectedExamId);
    if (Number.isNaN(examNumericId)) {
      setError('Identificador de examen invalido.');
      setResults([]);
      return;
    }

    load(examNumericId);
  }, [selectedExamId, token]);

  function handleSelectExam(event: Event) {
    const value = (event.target as HTMLSelectElement).value;
    setSelectedExamId(value);
  }

  async function handleImport(event: Event) {
    event.preventDefault();
    if (!selectedExamId) {
      setError('Selecciona un examen antes de importar.');
      return;
    }

    const examNumericId = Number(selectedExamId);
    if (Number.isNaN(examNumericId)) {
      setError('Identificador de examen invalido.');
      return;
    }

    const file = fileInputRef.current?.files?.[0];
    if (!file) {
      setError('Selecciona un archivo PDF para importar.');
      return;
    }

    setImporting(true);
    setError(null);

    try {
      const importSummary = await importOfficialResults(examNumericId, file, token);
      setSummary(importSummary);
      setFeedback(
        `Importacion completada: ${importSummary.imported_results}/${importSummary.total_rows} filas registradas.`,
      );
      const refreshed = await getOfficialResults(examNumericId, token);
      setResults(refreshed);
    } catch (err) {
      setSummary(null);
      setFeedback(null);
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setImporting(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  }

  return (
    <section className="mt-12">
      <h2 className="text-2xl font-bold text-purple-300 mb-4">Resultados oficiales</h2>

      <label className="block mb-4">
        <span className="block font-semibold mb-2">Selecciona un examen</span>
        <select
          value={selectedExamId}
          onChange={handleSelectExam}
          className="w-full max-w-sm px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white"
        >
          <option value="">-- Escoge un examen --</option>
          {exams.map((exam) => (
            <option key={exam.id} value={exam.id}>
              {exam.name} (ID: {exam.id})
            </option>
          ))}
        </select>
      </label>

      {error && (
        <p className="text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md mb-4">{error}</p>
      )}
      {feedback && (
        <p className="text-green-400 bg-green-400/10 border border-green-500 p-4 rounded-md mb-4">{feedback}</p>
      )}

      {summary && (
        <p className="text-xs text-gray-300 mb-4">
          Resumen: {summary.imported_results}/{summary.total_rows} filas guardadas, usuarios actualizados: {summary.updated_users}.
        </p>
      )}

      <form className="flex flex-col gap-3 md:flex-row md:items-center mb-6" onSubmit={handleImport}>
        <label className="flex flex-col gap-1 text-sm">
          <span>Importar PDF de resultados</span>
          <input
            ref={fileInputRef}
            type="file"
            accept="application/pdf"
            className="px-3 py-2 rounded border border-dashed border-[#555] bg-[#1f2229] text-white"
            disabled={!selectedExamId || importing}
          />
        </label>
        <button
          type="submit"
          className="py-2 px-4 rounded bg-blue-600 hover:bg-blue-700 transition-colors cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed"
          disabled={!selectedExamId || importing}
        >
          {importing ? 'Importando...' : 'Importar PDF'}
        </button>
      </form>

      {!selectedExamId && <p>Selecciona un examen para ver los resultados oficiales.</p>}

      {selectedExamId && loading && <p>Cargando resultados oficiales...</p>}

      {selectedExamId && !loading && results.length === 0 && !error && (
        <p>No hay resultados oficiales importados para este examen.</p>
      )}

      {selectedExamId && !loading && results.length > 0 && (
        <div className="bg-[#20232a] border border-[#333] rounded-lg p-4">
          <h3 className="text-lg font-semibold text-purple-200 mb-3">
            Listado oficial ({selectedExamName})
          </h3>
          <div className="overflow-x-auto">
            <table className="min-w-full text-sm text-left">
              <thead className="bg-[#2a2d33] text-gray-300">
                <tr>
                  <th className="px-3 py-2">DNI PDF</th>
                  <th className="px-3 py-2">Nombre</th>
                  <th className="px-3 py-2">Apellidos</th>
                  <th className="px-3 py-2">Usuario asociado</th>
                  <th className="px-3 py-2">Creado</th>
                </tr>
              </thead>
              <tbody>
                {results.slice(0, 100).map((result) => {
                  const user = result.user;
                  const surname = [result.apellido_1, result.apellido_2].filter(Boolean).join(' ');
                  const associatedUser = user
                    ? `${user.name} ${user.surname} (${user.dni}${user.email ? ` · ${user.email}` : ''})`
                    : 'Sin usuario enlazado';
                  return (
                    <tr key={result.id} className="odd:bg-[#1c1f26]">
                      <td className="px-3 py-2 font-mono text-xs">{result.dni_masked}</td>
                      <td className="px-3 py-2">{result.nombre}</td>
                      <td className="px-3 py-2">{surname}</td>
                      <td className="px-3 py-2">{associatedUser}</td>
                      <td className="px-3 py-2 text-xs text-gray-400">
                        {new Date(result.created_at).toLocaleString()}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
            {results.length > 100 && (
              <p className="text-xs text-gray-400 mt-2">
                Mostrando 100 de {results.length} resultados importados.
              </p>
            )}
          </div>
        </div>
      )}
    </section>
  );
}
