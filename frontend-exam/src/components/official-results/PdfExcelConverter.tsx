import { useRef, useState } from 'preact/hooks';
import {
  uploadPdfExcelJob,
  processPdfExcelJob,
  downloadPdfExcelResult,
  type ProcessPdfExcelResponse,
} from '../../services/pdfExcelService';

interface PdfExcelConverterProps {
  examId: string;
  token: string;
  onError?: (msg: string) => void;
}

export default function PdfExcelConverter({ examId, token, onError }: PdfExcelConverterProps) {
  const [uploading, setUploading] = useState(false);
  const [processing, setProcessing] = useState(false);
  const [uploadedCount, setUploadedCount] = useState<number | null>(null);
  const [result, setResult] = useState<ProcessPdfExcelResponse | null>(null);
  const [downloadReady, setDownloadReady] = useState(false);
  const [inlineError, setInlineError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Surface hard errors via the onError prop when provided, else inline.
  function reportError(message: string) {
    if (onError) {
      onError(message);
    } else {
      setInlineError(message);
    }
  }

  function clearError() {
    setInlineError(null);
    if (onError) onError('');
  }

  async function handleUpload(event: Event) {
    event.preventDefault();
    if (!examId) {
      reportError('Selecciona un examen antes de subir PDFs.');
      return;
    }

    const examNumericId = Number(examId);
    if (Number.isNaN(examNumericId)) {
      reportError('Identificador de examen inválido.');
      return;
    }

    const fileList = fileInputRef.current?.files;
    const files = fileList ? Array.from(fileList) : [];
    if (files.length === 0) {
      reportError('Selecciona al menos un PDF para subir.');
      return;
    }

    setUploading(true);
    clearError();
    // A fresh upload invalidates any previous processing result.
    setResult(null);
    setDownloadReady(false);

    try {
      const res = await uploadPdfExcelJob(examNumericId, files, token);
      setUploadedCount(res.count);
    } catch (err) {
      setUploadedCount(null);
      reportError(err instanceof Error ? err.message : String(err));
    } finally {
      setUploading(false);
      // Clear the input (matches OfficialResultImport) so a second click does
      // not silently re-upload the same batch.
      if (fileInputRef.current) fileInputRef.current.value = '';
    }
  }

  async function handleProcess() {
    const examNumericId = Number(examId);
    if (Number.isNaN(examNumericId)) {
      reportError('Identificador de examen inválido.');
      return;
    }

    setProcessing(true);
    clearError();

    try {
      const res = await processPdfExcelJob(examNumericId, token);
      setResult(res);
      setDownloadReady(res.ready);
    } catch (err) {
      setResult(null);
      setDownloadReady(false);
      reportError(err instanceof Error ? err.message : String(err));
    } finally {
      setProcessing(false);
    }
  }

  async function handleDownload() {
    const examNumericId = Number(examId);
    if (Number.isNaN(examNumericId)) {
      reportError('Identificador de examen inválido.');
      return;
    }
    try {
      await downloadPdfExcelResult(examNumericId, token);
    } catch (err) {
      reportError(err instanceof Error ? err.message : String(err));
    }
  }

  if (!examId) return null;

  const hasUpload = uploadedCount !== null;

  return (
    <div className="space-y-4 mb-6 bg-[#1f2229] p-4 rounded-md border border-[#444]">
      <h3 className="text-brand-pink font-semibold text-sm">
        Generar Excel desde PDFs (listas de admitidos)
      </h3>

      <form className="flex flex-col gap-3 md:flex-row md:items-center" onSubmit={handleUpload}>
        <label className="flex flex-col gap-1 text-sm">
          <span className="text-gray-300">Selecciona uno o varios PDFs</span>
          <input
            ref={fileInputRef}
            type="file"
            multiple
            accept=".pdf,application/pdf"
            className="px-3 py-2 rounded border border-dashed border-brand-pink-soft bg-[#1f2229] text-white cursor-pointer focus:outline-none focus:ring-2 focus:ring-brand-pink"
            disabled={uploading || processing}
          />
        </label>
        <button
          type="submit"
          className="py-2 px-5 rounded mt-5 md:mt-6 md:ml-5 font-semibold bg-brand-blue text-white shadow-[0_10px_30px_rgba(15,153,188,0.25)] hover:bg-[#12b2d4] transition-colors cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed"
          disabled={uploading || processing}
        >
          {uploading ? 'Subiendo...' : 'Subir PDFs'}
        </button>
      </form>

      {hasUpload && (
        <p className="text-sm text-green-400">{uploadedCount} PDFs subidos.</p>
      )}

      <div className="flex flex-wrap items-center gap-3">
        <button
          type="button"
          onClick={handleProcess}
          className="py-2 px-5 rounded font-semibold bg-brand-pink text-white hover:bg-brand-pink/90 transition-colors cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed"
          disabled={!hasUpload || uploading || processing}
        >
          {processing ? 'Procesando...' : 'Procesar'}
        </button>

        {downloadReady && (
          <button
            type="button"
            onClick={handleDownload}
            className="py-2 px-5 rounded font-semibold bg-brand-yellow text-[#14161d] hover:bg-yellow-400 transition-colors cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed"
            disabled={uploading || processing}
          >
            Descargar Excel
          </button>
        )}
      </div>

      {result && (
        <div className="space-y-2 text-sm">
          {result.already_generated ? (
            <p className="text-brand-blue">
              El Excel ya estaba generado. Pulsa «Descargar Excel».
            </p>
          ) : (
            <p className="text-brand-blue">
              Registros totales: {result.total_records}
              {result.expected_count !== null && ` / esperados: ${result.expected_count}`}
            </p>
          )}
          {result.warnings.length > 0 && (
            <div className="text-brand-blue bg-brand-blue/10 border border-brand-blue/40 p-3 rounded-md">
              <p className="font-semibold mb-1">Avisos:</p>
              <ul className="list-disc list-inside space-y-0.5">
                {result.warnings.map((warning, idx) => (
                  <li key={idx}>{warning}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}

      {!onError && inlineError && (
        <p className="text-red-500 bg-red-500/10 border border-red-500 p-3 rounded-md text-sm">
          {inlineError}
        </p>
      )}
    </div>
  );
}
