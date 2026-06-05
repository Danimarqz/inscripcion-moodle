// PDF -> Excel converter service. Exam-scoped: the backend derives the S3
// prefix as "exam-<exam_id>" from the URL param, so the frontend never sends a
// free-text prefix. Mirrors the raw-fetch + FormData + Authorization Bearer
// idioms used by importOfficialResults / downloadOfficialResultsTemplate.

export interface UploadPdfExcelResponse {
  files: string[];
  count: number;
}

export interface ProcessPdfExcelResponse {
  total_records: number;
  expected_count: number | null;
  warnings: string[];
  pdfs_processed: string[];
  ready: boolean;
  // True when the backend served an existing output without re-running the
  // Lambda; in that case total_records is not a real count.
  already_generated: boolean;
}

// Mirror importOfficialResults: parse {"detail": "..."} JSON first, fall back
// to plain text, else a default message.
async function extractError(response: Response, fallback: string): Promise<string> {
  const text = await response.text().catch(() => '');
  let message = fallback;
  try {
    const json = JSON.parse(text) as Record<string, unknown>;
    if (typeof json.detail === 'string') message = json.detail;
  } catch {
    if (text.trim()) message = text.trim();
  }
  return message;
}

export async function uploadPdfExcelJob(
  examId: number,
  files: File[],
  token: string,
): Promise<UploadPdfExcelResponse> {
  const formData = new FormData();
  for (const file of files) {
    formData.append('files', file);
  }

  // FormData handling is special, fetch handles content-type boundary
  const API_URL = import.meta.env.PUBLIC_API_URL;
  const response = await fetch(
    `${API_URL}/admin/exams/${examId}/results/pdf-excel/upload`,
    {
      method: 'POST',
      credentials: 'include',
      headers: { Authorization: `Bearer ${token}` },
      body: formData,
    },
  );

  if (!response.ok) {
    throw new Error(await extractError(response, 'Error subiendo los PDFs'));
  }

  return response.json() as Promise<UploadPdfExcelResponse>;
}

export async function processPdfExcelJob(
  examId: number,
  token: string,
): Promise<ProcessPdfExcelResponse> {
  const API_URL = import.meta.env.PUBLIC_API_URL;
  const response = await fetch(
    `${API_URL}/admin/exams/${examId}/results/pdf-excel/process`,
    {
      method: 'POST',
      credentials: 'include',
      headers: { Authorization: `Bearer ${token}` },
    },
  );

  if (!response.ok) {
    throw new Error(await extractError(response, 'Error procesando los PDFs'));
  }

  return response.json() as Promise<ProcessPdfExcelResponse>;
}

export async function downloadPdfExcelResult(examId: number, token: string): Promise<void> {
  const API_URL = import.meta.env.PUBLIC_API_URL;
  const response = await fetch(
    `${API_URL}/admin/exams/${examId}/results/pdf-excel/download`,
    {
      method: 'GET',
      credentials: 'include',
      headers: { Authorization: `Bearer ${token}` },
    },
  );
  if (!response.ok) {
    throw new Error(await extractError(response, 'Error descargando el Excel'));
  }
  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  // Match the backend Content-Disposition name (exam-<id>.xlsx).
  a.download = `exam-${examId}.xlsx`;
  a.click();
  URL.revokeObjectURL(url);
}
