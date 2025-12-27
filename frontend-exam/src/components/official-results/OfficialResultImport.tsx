import { useEffect, useRef, useState } from 'preact/hooks';
import { importOfficialResults } from '../../services/adminService';
import type { ImportOfficialResultsSummary } from '../../types/exam';

interface OfficialResultImportProps {
  examId: string;
  token: string;
  hasResults: boolean;
  onSuccess: (summary: ImportOfficialResultsSummary) => void;
  onError: (error: string) => void;
}

export default function OfficialResultImport({
  examId,
  token,
  hasResults,
  onSuccess,
  onError,
}: OfficialResultImportProps) {
  const [importing, setImporting] = useState(false);
  const [replaceExisting, setReplaceExisting] = useState(!hasResults);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Update default when hasResults changes
  useEffect(() => {
    setReplaceExisting(!hasResults);
  }, [hasResults]);

  async function handleImport(event: Event) {
    event.preventDefault();
    if (!examId) {
      onError('Selecciona un examen antes de importar.');
      return;
    }

    const examNumericId = Number(examId);
    if (Number.isNaN(examNumericId)) {
      onError('Identificador de examen inválido.');
      return;
    }

    const file = fileInputRef.current?.files?.[0];
    if (!file) {
      onError('Selecciona un archivo Excel para importar.');
      return;
    }

    setImporting(true);
    onError(''); // Clear error

    try {
      const summary = await importOfficialResults(examNumericId, file, token, replaceExisting);
      onSuccess(summary);
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err));
    } finally {
      setImporting(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  }

  if (!examId) return null;

  return (
    <div className="space-y-4 mb-6">
      <form className="flex flex-col gap-3 md:flex-row md:items-center" onSubmit={handleImport}>
        <label className="flex flex-col gap-1 text-sm">
          <span className="text-brand-pink font-semibold">Importar Excel de resultados</span>
          <input
            ref={fileInputRef}
            type="file"
            accept=".xlsx, .xls, application/vnd.openxmlformats-officedocument.spreadsheetml.sheet, application/vnd.ms-excel"
            className="px-3 py-2 rounded border border-dashed border-brand-pink-soft bg-[#1f2229] text-white cursor-pointer focus:outline-none focus:ring-2 focus:ring-brand-pink"
            disabled={importing}
          />
        </label>
        <button
          type="submit"
          className="py-2 px-5 rounded mt-5 md:mt-6 md:ml-5 font-semibold bg-brand-blue text-white shadow-[0_10px_30px_rgba(15,153,188,0.25)] hover:bg-[#12b2d4] transition-colors cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed"
          disabled={importing}
        >
          {importing ? 'Importando...' : 'Importar Excel'}
        </button>
      </form>

      <label className="flex items-center gap-2 cursor-pointer group">
        <input
          type="checkbox"
          checked={replaceExisting}
          onChange={(e) => setReplaceExisting((e.target as HTMLInputElement).checked)}
          className="w-4 h-4 rounded border-[#444] bg-[#1f2229] text-brand-blue focus:ring-brand-blue"
        />
        <span className="text-sm text-gray-300 group-hover:text-white transition-colors">
          Reemplazar resultados existentes para este examen
        </span>
      </label>
    </div>
  );
}
