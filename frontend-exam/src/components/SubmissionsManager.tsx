import { useCallback, useEffect, useMemo, useState } from 'preact/hooks';

import type { AdminSubmission, AdminSubmissionsResponse, Exam, QuestionEdit } from '../types/exam';
import {
  deleteSubmissionAttempt,
  downloadSubmissionEmails,
  fetchSubmissionEmailList,
  getExamById,
  getExamSubmissions,
  sendSubmissionEmails,
  syncMoodleUsers,
  type SubmissionEmailAttachmentPayload,
  type SendSubmissionEmailsPayload,
  updateSubmissionAttempt,
} from '../services/adminService';
import { normalizeDni, validateDniNie } from '../utils/validation';

import ExamSelector from './submissions/ExamSelector';
import SubmissionFilters from './submissions/SubmissionFilters';
import SubmissionFilterActions from './submissions/SubmissionFilterActions';
import SubmissionStats from './submissions/SubmissionStats';
import EmailComposer from './submissions/EmailComposer';
import PaginationSettings from './submissions/PaginationSettings';
import PaginationControls from './submissions/PaginationControls';
import SubmissionList from './submissions/SubmissionList';
import ConfirmModal from './modals/ConfirmModal';
import { ANSWER_OPTIONS } from './submissions/types';
import type { AnswerOption, EditingState, SubmissionOrderBy, SubmissionOrderDir } from './submissions/types';

type SubmissionStats = {
  totalSubmissions: number;
  averageScore: number | null;
};

const INITIAL_SUBMISSION_STATS: SubmissionStats = {
  totalSubmissions: 0,
  averageScore: null,
};

interface SubmissionsManagerProps {
  exams: Exam[];
  token: string;
}

function isValidEmail(value: string): boolean {
  return /^[\w-.]+@([\w-]+\.)+[\w-]{2,}$/i.test(value.trim());
}

export default function SubmissionsManager({ exams, token }: SubmissionsManagerProps) {
  const [selectedExamId, setSelectedExamId] = useState<string>('');
  const [questions, setQuestions] = useState<QuestionEdit[]>([]);
  const [submissions, setSubmissions] = useState<AdminSubmission[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [editing, setEditing] = useState<EditingState | null>(null);
  const [saving, setSaving] = useState<boolean>(false);
  const [filterMoodleUsers, setFilterMoodleUsers] = useState(false);
  const [syncingMoodle, setSyncingMoodle] = useState(false);
  const [syncMessage, setSyncMessage] = useState<string | null>(null);
  const [syncError, setSyncError] = useState<string | null>(null);
  const [downloadingEmails, setDownloadingEmails] = useState(false);
  const [downloadMessage, setDownloadMessage] = useState<string | null>(null);
  const [downloadError, setDownloadError] = useState<string | null>(null);
  const [emailComposerOpen, setEmailComposerOpen] = useState(false);
  const [emailCandidates, setEmailCandidates] = useState<{ email: string; selected: boolean }[]>([]);
  const [emailLoading, setEmailLoading] = useState(false);
  const [emailComposeModalError, setEmailComposeModalError] = useState<string | null>(null);
  const [emailComposeMessage, setEmailComposeMessage] = useState<string | null>(null);
  const [emailSubject, setEmailSubject] = useState('');
  const [emailBody, setEmailBody] = useState('');
  const [emailAttachments, setEmailAttachments] = useState<File[]>([]);
  const [emailSending, setEmailSending] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const [submissionStats, setSubmissionStats] = useState<SubmissionStats>(() => ({
    ...INITIAL_SUBMISSION_STATS,
  }));
  const [needsStats, setNeedsStats] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [debouncedSearchTerm, setDebouncedSearchTerm] = useState('');
  const [orderBy, setOrderBy] = useState<SubmissionOrderBy>('submitted_at');
  const [orderDir, setOrderDir] = useState<SubmissionOrderDir>('desc');
  const [pageLimit, setPageLimit] = useState(25);
  const [pageInput, setPageInput] = useState(1);
  const [subDeleteId, setSubDeleteId] = useState<number | null>(null);
  const resetFilters = useCallback(() => {
    setCurrentPage(1);
    setNeedsStats(true);
    setSubmissionStats({ ...INITIAL_SUBMISSION_STATS });
    setSubmissions([]);
    setQuestions([]);
    setEditing(null);
    setFeedback(null);
    setError(null);
    setSearchTerm('');
    setOrderBy('submitted_at');
    setOrderDir('desc');
    setPageLimit(25);
    setPageInput(1);
  }, []);
  const handleSearchInput = (value: string) => {
    setSearchTerm(value);
    setCurrentPage(1);
  };
  const handleOrderByChange = (value: SubmissionOrderBy) => {
    setOrderBy(value);
    setCurrentPage(1);
  };
  const handleOrderDirChange = (value: SubmissionOrderDir) => {
    setOrderDir(value);
    setCurrentPage(1);
  };
  const handleLimitChange = (value: number) => {
    if (value <= 0) return;
    setPageLimit(value);
    setCurrentPage(1);
    setPageInput(1);
  };
  const handlePageInputChange = (value: number) => {
    setPageInput(value);
  };
  const goToPage = () => {
    const desired = Math.max(1, Math.min(totalPages, Math.floor(pageInput)));
    setCurrentPage(desired);
  };
  const handleFilterMoodleUsersChange = (value: boolean) => {
    setFilterMoodleUsers(value);
    setCurrentPage(1);
    setNeedsStats(true);
    setSubmissionStats({ ...INITIAL_SUBMISSION_STATS });
  };

  const selectedExamName = useMemo(() => {
    const numericId = Number(selectedExamId);
    return exams.find((exam) => exam.id === numericId)?.name ?? '';
  }, [exams, selectedExamId]);
  const selectedEmailCount = emailCandidates.filter((candidate) => candidate.selected).length;

  async function loadSubmissionsPage(
    examNumericId: number,
    page: number,
    limit: number,
    includeStats = false,
    search = '',
    orderBy: SubmissionOrderBy = 'submitted_at',
    orderDir: SubmissionOrderDir = 'desc',
    moodleSynced?: boolean,
  ): Promise<AdminSubmissionsResponse | null> {
    const response = await getExamSubmissions(examNumericId, token, {
      limit,
      offset: (page - 1) * limit,
      firstLoad: includeStats,
      search,
      orderBy,
      orderDir,
      moodleSynced,
    });
    const knownTotal = includeStats && response.stats_included
      ? response.total_submissions
      : submissionStats.totalSubmissions;
    const totalPages = Math.max(1, Math.ceil(Math.max(knownTotal, 1) / limit));
    if (page > totalPages) {
      setCurrentPage(totalPages);
      return null;
    }

    setSubmissions(response.submissions);
    if (includeStats && response.stats_included) {
      setSubmissionStats({
        totalSubmissions: response.total_submissions,
        averageScore: response.average_score,
      });
      setNeedsStats(false);
    }
    return response;
  }

  const { totalSubmissions, averageScore } = submissionStats;
  const effectiveLimit = pageLimit > 0 ? pageLimit : 1;
  const totalPages = Math.max(1, Math.ceil(totalSubmissions / effectiveLimit));

  const handlePrevPage = () => setCurrentPage((prev) => Math.max(1, prev - 1));
  const handleNextPage = () => setCurrentPage((prev) => Math.min(totalPages, prev + 1));

  useEffect(() => {
    async function loadData(examNumericId: number) {
      setLoading(true);
      setError(null);
      setFeedback(null);
      setEditing(null);
      try {
        const pageResponse = await loadSubmissionsPage(
          examNumericId,
          currentPage,
          pageLimit,
          needsStats,
          debouncedSearchTerm,
          orderBy,
          orderDir,
          filterMoodleUsers ? true : undefined,
        );
        if (!pageResponse) {
          return;
        }
        if (needsStats) {
          const examData = await getExamById(examNumericId, token);
          setQuestions(examData.questions ?? []);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
        setSubmissions([]);
        setQuestions([]);
        setSubmissionStats({ ...INITIAL_SUBMISSION_STATS });
      } finally {
        setLoading(false);
      }
    }

    if (!selectedExamId) {
      setSubmissions([]);
      setQuestions([]);
      setEditing(null);
      setFeedback(null);
      setSubmissionStats({ ...INITIAL_SUBMISSION_STATS });
      setNeedsStats(true);
      return;
    }

    const examNumericId = Number(selectedExamId);
    if (Number.isNaN(examNumericId)) {
      setError('Identificador de examen invalido.');
      setSubmissions([]);
      setQuestions([]);
      setSubmissionStats({ ...INITIAL_SUBMISSION_STATS });
      return;
    }

    loadData(examNumericId);
  }, [
    selectedExamId,
    token,
    currentPage,
    debouncedSearchTerm,
    orderBy,
    orderDir,
    pageLimit,
    filterMoodleUsers,
  ]);

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedSearchTerm(searchTerm);
    }, 500);
    return () => clearTimeout(handler);
  }, [searchTerm]);

  useEffect(() => {
    resetFilters();
  }, [selectedExamId, resetFilters]);

  useEffect(() => {
    setDownloadMessage(null);
    setDownloadError(null);
  }, [searchTerm, filterMoodleUsers]);

  useEffect(() => {
    setPageInput(currentPage);
  }, [currentPage]);

  function startEditing(submission: AdminSubmission) {
    const initialAnswers: Record<number, AnswerOption> = {};
    submission.answers.forEach((answer) => {
      const normalizedAnswer = (answer.answer ?? '').toUpperCase();
      const castAnswer = normalizedAnswer as AnswerOption;
      initialAnswers[answer.question_id] = ANSWER_OPTIONS.includes(castAnswer)
        ? castAnswer
        : ANSWER_OPTIONS[0];
    });

    const user = submission.user;
    setEditing({
      submissionId: submission.id,
      name: submission.name || user?.name || '',
      surname: submission.surname || user?.surname || '',
      email: submission.email ?? user?.email ?? '',
      dni: submission.dni || user?.dni || '',
      answers: initialAnswers,
    });
    setFeedback(null);
    setError(null);
  }

  async function handleDelete(submissionId: number) {
    setSubDeleteId(submissionId);
  }

  const confirmSubDelete = async () => {
    if (subDeleteId === null || !selectedExamId) return;
    try {
      await deleteSubmissionAttempt(subDeleteId, token);
      if (selectedExamId) {
        const examNumericId = Number(selectedExamId);
          if (!Number.isNaN(examNumericId)) {
            await loadSubmissionsPage(
              examNumericId,
              currentPage,
              pageLimit,
              needsStats,
              searchTerm,
              orderBy,
              orderDir,
              filterMoodleUsers ? true : undefined,
            );
          }
      }
      setFeedback('Intento eliminado correctamente.');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubDeleteId(null);
    }
  };

  function updateEditingField<K extends keyof Omit<EditingState, 'answers' | 'submissionId'>>(
    field: K,
    value: EditingState[K],
  ) {
    if (!editing) return;
    setEditing({ ...editing, [field]: value });
  }

  function updateAnswer(questionId: number, value: string) {
    if (!editing) return;
    const upper = value.toUpperCase();
    const normalized = upper as AnswerOption;
    if (!ANSWER_OPTIONS.includes(normalized)) return;

    setEditing({
      ...editing,
      answers: {
        ...editing.answers,
        [questionId]: normalized,
      },
    });
  }

  async function handleSave() {
    if (!editing) return;

    setError(null);
    setFeedback(null);

    const trimmedName = editing.name.trim();
    const trimmedSurname = editing.surname.trim();
    const trimmedEmail = editing.email.trim().toLowerCase();
    const normalizedDni = normalizeDni(editing.dni);

    if (!trimmedName) {
      setError('El nombre es obligatorio.');
      return;
    }
    if (!trimmedSurname) {
      setError('Los apellidos son obligatorios.');
      return;
    }
    if (!isValidEmail(trimmedEmail)) {
      setError('Introduce un email valido.');
      return;
    }
    if (!validateDniNie(normalizedDni)) {
      setError('Introduce un DNI o NIE valido.');
      return;
    }

    const answersPayload = questions
      .filter((question): question is QuestionEdit & { id: number } => typeof question.id === 'number')
      .map((question) => ({
        question_id: question.id,
        answer: (editing.answers[question.id] || 'A').toUpperCase(),
      }));

    if (answersPayload.length !== questions.length) {
      setError('No se pudieron obtener todas las preguntas del examen.');
      return;
    }

    setSaving(true);
    try {
      const updated = await updateSubmissionAttempt(
        editing.submissionId,
        {
          name: trimmedName,
          surname: trimmedSurname,
          email: trimmedEmail,
          dni: normalizedDni,
          answers: answersPayload,
        },
        token,
      );

      const examNumericId = Number(selectedExamId);
        if (selectedExamId && !Number.isNaN(examNumericId)) {
          await loadSubmissionsPage(
            examNumericId,
            currentPage,
            pageLimit,
            needsStats,
            searchTerm,
            orderBy,
            orderDir,
            filterMoodleUsers ? true : undefined,
          );
        }

      setEditing(null);
      setFeedback('Intento actualizado correctamente.');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  async function handleSyncMoodleUsers() {
    setSyncMessage(null);
    setSyncError(null);
    setSyncingMoodle(true);
    try {
      const result = await syncMoodleUsers(token);
      setSyncMessage(
        `Sincronizados ${result.synced} de ${result.checked} usuarios (fallidos: ${result.failed}).`,
      );
    } catch (err) {
      setSyncError(err instanceof Error ? err.message : String(err));
    } finally {
      setSyncingMoodle(false);
    }
  }

  async function handleDownloadEmails() {
    if (!selectedExamId) {
      return;
    }
    setDownloadMessage(null);
    setDownloadError(null);
    setDownloadingEmails(true);
    try {
      const content = await downloadSubmissionEmails(Number(selectedExamId), token, {
        search: searchTerm,
        moodleSynced: filterMoodleUsers ? true : undefined,
      });
      const lines = content
        .split(/\r?\n/)
        .map((line) => line.trim())
        .filter((line) => line !== '');
      if (lines.length === 0) {
        setDownloadError('No hay correos para descargar con los filtros actuales.');
        return;
      }
      const blob = new Blob([lines.join('\n')], { type: 'text/plain;charset=utf-8' });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = `emails_${selectedExamId}.txt`;
      document.body.appendChild(anchor);
      anchor.click();
      document.body.removeChild(anchor);
      URL.revokeObjectURL(url);
      setDownloadMessage(`Descargados ${lines.length} emails.`);
    } catch (err) {
      setDownloadError(err instanceof Error ? err.message : String(err));
    } finally {
      setDownloadingEmails(false);
    }
  }

  const readFileAsBase64 = (file: File): Promise<string> =>
    new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => {
        const result = reader.result as string;
        const commaIndex = result.indexOf(',');
        resolve(commaIndex >= 0 ? result.slice(commaIndex + 1) : result);
      };
      reader.onerror = () => reject(reader.error);
      reader.readAsDataURL(file);
    });

  const closeEmailComposer = () => {
    setEmailComposerOpen(false);
    setEmailComposeModalError(null);
    setEmailCandidates([]);
    setEmailAttachments([]);
  };

  const handleComposeEmails = async () => {
    if (!selectedExamId) {
      return;
    }
    setEmailComposeModalError(null);
    setEmailComposeMessage(null);
    setEmailComposerOpen(true);
    setEmailLoading(true);
    setEmailCandidates([]);
    setEmailAttachments([]);
    setEmailSubject(selectedExamName ? `Comunicado sobre ${selectedExamName}` : 'Comunicado oficial');
    setEmailBody('');
    try {
      const emails = await fetchSubmissionEmailList(Number(selectedExamId), token, {
        search: searchTerm,
        orderBy,
        orderDir,
        moodleSynced: filterMoodleUsers ? true : undefined,
      });
      setEmailCandidates(emails.map((address) => ({ email: address, selected: true })));
    } catch (err) {
      setEmailComposeModalError(err instanceof Error ? err.message : String(err));
    } finally {
      setEmailLoading(false);
    }
  };

  const toggleEmailRecipient = (index: number) => {
    setEmailCandidates((prev) =>
      prev.map((item, idx) => (idx === index ? { ...item, selected: !item.selected } : item)),
    );
  };

  const handleAttachmentChange = (files: FileList | null) => {
    if (!files) {
      return;
    }
    setEmailAttachments((prev) => [...prev, ...Array.from(files)]);
  };

  const handleRemoveAttachment = (index: number) => {
    setEmailAttachments((prev) => prev.filter((_, idx) => idx !== index));
  };

  const handleSendEmails = async () => {
    if (!selectedExamId) {
      return;
    }
    const recipients = emailCandidates.filter((candidate) => candidate.selected).map((item) => item.email);
    if (recipients.length === 0) {
      setEmailComposeModalError('Selecciona al menos un destinatario.');
      return;
    }
    const trimmedBody = emailBody.trim();
    if (!trimmedBody) {
      setEmailComposeModalError('El cuerpo del email no puede estar vacío.');
      return;
    }
    setEmailComposeModalError(null);
    setEmailSending(true);
    try {
      const attachments = await Promise.all(
        emailAttachments.map(async (file) => {
          const content = await readFileAsBase64(file);
          return {
            filename: file.name,
            content_type: file.type || 'application/octet-stream',
            content,
          } as SubmissionEmailAttachmentPayload;
        }),
      );
      const payload: SendSubmissionEmailsPayload = {
        exam_id: Number(selectedExamId),
        subject:
          emailSubject || (selectedExamName ? `Comunicado sobre ${selectedExamName}` : 'Comunicado oficial'),
        body: trimmedBody,
        recipients,
        search: searchTerm,
        order_by: orderBy,
        order_dir: orderDir,
        moodle_synced: filterMoodleUsers ? true : undefined,
        attachments: attachments.length > 0 ? attachments : undefined,
      };
      await sendSubmissionEmails(payload, token);
      setEmailComposeMessage(`Correo enviado a ${recipients.length} destinatarios.`);
      closeEmailComposer();
    } catch (err) {
      setEmailComposeModalError(err instanceof Error ? err.message : String(err));
    } finally {
      setEmailSending(false);
    }
  };

  return (
    <section className="mt-8 space-y-6">
      <ExamSelector
        exams={exams}
        selectedExamId={selectedExamId}
        onSelectExam={(value) => setSelectedExamId(value)}
        onSyncMoodleUsers={handleSyncMoodleUsers}
        syncingMoodle={syncingMoodle}
        syncMessage={syncMessage}
        syncError={syncError}
        canSync={Boolean(selectedExamId && (totalSubmissions > 0 || filterMoodleUsers))}
      />

      <SubmissionFilters
        searchTerm={searchTerm}
        onSearchTermChange={handleSearchInput}
        orderBy={orderBy}
        onOrderByChange={handleOrderByChange}
        orderDir={orderDir}
        onOrderDirChange={handleOrderDirChange}
      />

      {selectedExamId && (totalSubmissions > 0 || filterMoodleUsers) && (
        <SubmissionFilterActions
          filterMoodleUsers={filterMoodleUsers}
          onFilterChange={handleFilterMoodleUsersChange}
          downloadingEmails={downloadingEmails}
          downloadMessage={downloadMessage}
          downloadError={downloadError}
          onDownloadEmails={handleDownloadEmails}
          onComposeEmails={handleComposeEmails}
          composingEmails={emailLoading || emailSending}
          composeMessage={emailComposeMessage}
          composeError={emailComposeModalError}
        />
      )}

      {error && (
        <p className="text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md mb-4">{error}</p>
      )}
      {feedback && (
        <p className="text-green-400 bg-green-400/10 border border-green-500 p-4 rounded-md mb-4">{feedback}</p>
      )}

      {selectedExamId && (
        <SubmissionStats
          totalSubmissions={totalSubmissions}
          averageScore={averageScore}
          needsStats={needsStats}
          selectedExamName={selectedExamName}
          loading={loading}
        />
      )}

      <PaginationSettings
        pageLimit={pageLimit}
        onLimitChange={handleLimitChange}
        pageInput={pageInput}
        onPageInputChange={handlePageInputChange}
        totalPages={totalPages}
        goToPage={goToPage}
      />

      {!selectedExamId && (
        <p className="text-sm text-brand-blue">Selecciona un examen para ver los intentos disponibles.</p>
      )}

      {selectedExamId && loading && <p className="text-brand-yellow">Cargando intentos...</p>}

      {selectedExamId && !loading && totalSubmissions === 0 && !error && (
        <p className="text-sm text-brand-pink">No hay intentos para este examen.</p>
      )}

      {selectedExamId && !loading && totalSubmissions > 0 && submissions.length === 0 && !error && (
        <p className="text-sm text-brand-blue">Esta pagina no contiene intentos.</p>
      )}

      {selectedExamId && !loading && totalSubmissions > 0 && (
        <PaginationControls
          currentPage={currentPage}
          totalPages={totalPages}
          onPrev={handlePrevPage}
          onNext={handleNextPage}
        />
      )}

      {selectedExamId && !loading && submissions.length > 0 && (
        <SubmissionList
          submissions={submissions}
          editing={editing}
          questions={questions}
          selectedExamName={selectedExamName}
          answerOptions={ANSWER_OPTIONS}
          onStartEditing={startEditing}
          onCancelEditing={() => setEditing(null)}
          onDelete={handleDelete}
          onUpdateField={updateEditingField}
          onUpdateAnswer={updateAnswer}
          onSave={handleSave}
          saving={saving}
        />
      )}

      {selectedExamId && !loading && totalSubmissions > 0 && (
        <PaginationControls
          currentPage={currentPage}
          totalPages={totalPages}
          onPrev={handlePrevPage}
          onNext={handleNextPage}
        />
      )}
      {emailComposerOpen && (
        <EmailComposer
          loading={emailLoading}
          sending={emailSending}
          subject={emailSubject}
          body={emailBody}
          attachments={emailAttachments}
          candidates={emailCandidates}
          selectedCount={selectedEmailCount}
          error={emailComposeModalError}
          onSubjectChange={setEmailSubject}
          onBodyChange={setEmailBody}
          onAddAttachments={handleAttachmentChange}
          onRemoveAttachment={handleRemoveAttachment}
          onToggleRecipient={toggleEmailRecipient}
          onClose={closeEmailComposer}
          onSend={handleSendEmails}
        />
      )}

      <ConfirmModal
        isOpen={subDeleteId !== null}
        title="Eliminar Intento"
        message="¿Estás seguro de que quieres eliminar este intento de examen? Esta acción es irreversible."
        confirmText="Eliminar"
        isDanger={true}
        onConfirm={confirmSubDelete}
        onClose={() => setSubDeleteId(null)}
      />
    </section>
  );
}
