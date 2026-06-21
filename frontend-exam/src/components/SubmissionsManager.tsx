import { useEffect, useMemo, useState, useRef } from 'preact/hooks';

import type { AdminSubmission, Exam, QuestionEdit } from '../types/exam';
import {
  deleteSubmissionAttempt,
  downloadSubmissionEmails,
  downloadSubmissionsAnalysis,
  fetchSubmissionEmailList,
  getExamById,
  getExamQuestions,
  getExamSubmissions,
  getSubmission,
  sendSubmissionEmails,
  syncMoodleUsersStream,
  type SubmissionEmailAttachmentPayload,
  type SendSubmissionEmailsPayload,
  updateSubmissionAttempt,
} from '../services/adminService';
import { isValidEmail, normalizeDni, validateDniNie } from '../utils/validation';

import ExamSelector from './submissions/ExamSelector';
import SubmissionFilters from './submissions/SubmissionFilters';
import SubmissionFilterActions from './submissions/SubmissionFilterActions';
import SubmissionStats from './submissions/SubmissionStats';
import EmailComposer from './submissions/EmailComposer';
import PaginationSettings from './submissions/PaginationSettings';
import PaginationControls from './shared/PaginationControls';
import SubmissionList from './submissions/SubmissionList';
import ConfirmModal from './modals/ConfirmModal';
import { ANSWER_OPTIONS } from './submissions/types';
import type { AnswerOption, EditingState, SubmissionOrderBy, SubmissionOrderDir } from './submissions/types';

type SubmissionStats = {
  totalSubmissions: number;
  averageScore: number | null;
  averageScoreOfficial: number | null;
  groupAverageScore: number | null;
  groupAverageScoreOfficial: number | null;
  groupTotalSubmissions: number | null;
  groupExamNames: string[] | null;
};

const INITIAL_SUBMISSION_STATS: SubmissionStats = {
  totalSubmissions: 0,
  averageScore: null,
  averageScoreOfficial: null,
  groupAverageScore: null,
  groupAverageScoreOfficial: null,
  groupTotalSubmissions: null,
  groupExamNames: null,
};

interface SubmissionsManagerProps {
  exams: Exam[];
  token: string;
}

const MAX_ATTACHMENT_SIZE_BYTES = 5 * 1024 * 1024; // 5 MB

interface SubmissionsViewerProps {
  examId: number;
  selectedExamName: string;
  token: string;
}

function SubmissionsViewer({ examId, selectedExamName, token }: SubmissionsViewerProps) {
  const [questions, setQuestions] = useState<QuestionEdit[]>([]);
  const [submissions, setSubmissions] = useState<AdminSubmission[]>([]);
  const [loading, setLoading] = useState<boolean>(true); // Start loading immediately since we only mount when ID exists
  const [error, setError] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [editingStates, setEditingStates] = useState<Record<number, EditingState>>({});
  const [savingIds, setSavingIds] = useState<Record<number, boolean>>({});
  const [filterMoodleUsers, setFilterMoodleUsers] = useState(false);
  const [compareGroup, setCompareGroup] = useState(false);
  const [examConfig, setExamConfig] = useState<{
    maxScore?: number;
    secondaryMaxScores?: string;
    scoringMode?: string;
    pointsPerCorrect?: number;
  } | null>(null);

  const [downloadingEmails, setDownloadingEmails] = useState(false);
  const [downloadingAnalysis, setDownloadingAnalysis] = useState(false);
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
  const [filterType, setFilterType] = useState<string>('');
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  // Prevents stale fetch responses from overwriting a more recent one.
  const fetchRequestId = useRef(0);

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
  const handleFilterTypeChange = (value: string) => {
    setFilterType(value);
    setNeedsStats(true);
    setSubmissionStats({ ...INITIAL_SUBMISSION_STATS });
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

  const selectedEmailCount = emailCandidates.filter((candidate) => candidate.selected).length;
  const {
    totalSubmissions,
    averageScore,
    averageScoreOfficial,
    groupAverageScore,
    groupAverageScoreOfficial,
    groupTotalSubmissions,
    groupExamNames,
  } = submissionStats;
  const effectiveLimit = pageLimit > 0 ? pageLimit : 1;
  const totalPages = Math.max(1, Math.ceil(totalSubmissions / effectiveLimit));

  // Data Fetching Effect
  useEffect(() => {
    let mounted = true;
    const requestId = ++fetchRequestId.current;

    const fetchData = async () => {
      setLoading(true);
      setError(null);
      setFeedback(null);
      setEditingStates({});

      try {
        const response = await getExamSubmissions(examId, token, {
          limit: pageLimit,
          offset: (currentPage - 1) * pageLimit,
          firstLoad: needsStats,
          search: debouncedSearchTerm,
          orderBy,
          orderDir,
          moodleSynced: filterMoodleUsers ? true : undefined,
          type: filterType,
        });

        // Discard if unmounted or a newer request has already started.
        if (!mounted || requestId !== fetchRequestId.current) return;

        setSubmissions(response.submissions);

        if (needsStats) {
          if (response.stats_included) {
            setSubmissionStats({
              totalSubmissions: response.total_submissions ?? 0,
              averageScore: response.average_score ?? null,
              averageScoreOfficial: response.average_score_official ?? null,
              groupAverageScore: response.group_average_score ?? null,
              groupAverageScoreOfficial: response.group_average_score_official ?? null,
              groupTotalSubmissions: response.group_total_submissions ?? null,
              groupExamNames: response.group_exam_names ?? null,
            });
            setNeedsStats(false);
          } else if (response.submissions.length === 0) {
            setSubmissionStats(INITIAL_SUBMISSION_STATS);
            setNeedsStats(false);
          }
        }

        if (needsStats) {
          try {
            // Only exam config here (small). Questions are loaded lazily on edit.
            const examData = await getExamById(examId, token);
            if (mounted && requestId === fetchRequestId.current) {
              // Absolute mode: the effective max is points_per_correct * active
              // questions. We don't have questions loaded here, so fall back to
              // the stored max_score; the edit view recomputes precisely once
              // questions are fetched.
              const effectiveMax = examData.max_score;
              setExamConfig({
                maxScore: effectiveMax,
                secondaryMaxScores: examData.secondary_max_scores,
                scoringMode: examData.scoring_mode,
                pointsPerCorrect: examData.points_per_correct ?? undefined,
              });
            }
          } catch (qErr) {
            if (import.meta.env.DEV) {
              console.error('Failed to load exam config:', qErr);
            }
          }
        }
      } catch (err) {
        if (mounted && requestId === fetchRequestId.current) {
          setError(err instanceof Error ? err.message : String(err));
          setSubmissions([]);
          setSubmissionStats(INITIAL_SUBMISSION_STATS);
          setNeedsStats(false);
        }
      } finally {
        if (mounted && requestId === fetchRequestId.current) {
          setLoading(false);
        }
      }
    };

    fetchData();

    return () => {
      mounted = false;
    };
  }, [
    examId,
    token, 
    currentPage,
    debouncedSearchTerm,
    orderBy,
    orderDir,
    pageLimit,
    filterMoodleUsers,
    filterType,
    refreshTrigger,
  ]);

  // Debounce Search
  useEffect(() => {
    const handler = setTimeout(() => {
      if (searchTerm !== debouncedSearchTerm) {
        setDebouncedSearchTerm(searchTerm);
        setCurrentPage(1);
        setNeedsStats(true);
      }
    }, 500);
    return () => clearTimeout(handler);
  }, [searchTerm, debouncedSearchTerm]);
  
  // Reset for download UI on filter change
  useEffect(() => {
    setDownloadMessage(null);
    setDownloadError(null);
  }, [searchTerm, filterMoodleUsers]);

  // Sync page input
  useEffect(() => {
    setPageInput(currentPage);
  }, [currentPage]);


  // Handler Copies (Simplified references)
  async function startEditing(submission: AdminSubmission) {
    if (editingStates[submission.id]) return; // Already editing

    // Optional: set loading indicator for this specific item if we wanted to be fancy.
    // For now global feedback is okay-ish but multiple loading messages would race.
    // Let's use a local trick or just rely on the UI not blocking others.
    
    try {
      // Questions are needed only for the answer editor, so they're loaded the
      // first time an attempt is edited (in parallel with the submission), not
      // on every list open.
      const [fullSubmission, loadedQuestions] = await Promise.all([
        getSubmission(submission.id, token),
        questions.length === 0 ? getExamQuestions(examId, token) : Promise.resolve(questions),
      ]);
      if (questions.length === 0 && loadedQuestions.length > 0) {
        setQuestions(loadedQuestions);
        // Absolute mode: refine the displayed max now that questions are known.
        if (examConfig?.scoringMode === 'absolute' && examConfig.pointsPerCorrect != null) {
          const activeCount = loadedQuestions.filter(
            (q) => q.is_active !== false && q.is_cancelled !== true,
          ).length;
          setExamConfig((prev) =>
            prev ? { ...prev, maxScore: examConfig.pointsPerCorrect! * activeCount } : prev,
          );
        }
      }

      const initialAnswers: Record<number, AnswerOption | '-'> = {};
      if (fullSubmission.answers_data) {
        for (const [qid, val] of Object.entries(fullSubmission.answers_data)) {
          const normalizedAnswer = (val ?? '').toUpperCase();
          const castAnswer = normalizedAnswer as AnswerOption;
          initialAnswers[Number(qid)] = ANSWER_OPTIONS.includes(castAnswer) ? castAnswer : '-';
        }
      }
      const user = fullSubmission.user;
      
      setEditingStates((prev) => ({
        ...prev,
        [submission.id]: {
            submissionId: fullSubmission.id,
            name: fullSubmission.name || user?.name || '',
            surname: fullSubmission.surname || user?.surname || '',
            email: fullSubmission.email ?? user?.email ?? '',
            dni: fullSubmission.dni || user?.dni || '',
            answers: initialAnswers,
            merits: fullSubmission.merits ?? undefined,
        }
      }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  function cancelEditing(submissionId: number) {
    setEditingStates((prev) => {
        const next = { ...prev };
        delete next[submissionId];
        return next;
    });
  }

  async function handleDelete(submissionId: number) {
    setSubDeleteId(submissionId);
  }

  const confirmSubDelete = async () => {
    if (subDeleteId === null) return;
    try {
      await deleteSubmissionAttempt(subDeleteId, token);
      setNeedsStats(true); 
      setRefreshTrigger((prev) => prev + 1); 
      setFeedback('Intento eliminado correctamente.');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubDeleteId(null);
    }
  };

  function updateEditingField<K extends keyof Omit<EditingState, 'answers' | 'submissionId'>>(submissionId: number, field: K, value: EditingState[K]) {
    setEditingStates((prev) => {
        const current = prev[submissionId];
        if (!current) return prev;
        return {
            ...prev,
            [submissionId]: { ...current, [field]: value }
        };
    });
  }

  function updateAnswer(submissionId: number, questionId: number, value: string) {
    const upper = value.toUpperCase();
    if (upper !== '-' && !ANSWER_OPTIONS.includes(upper as AnswerOption)) return;

    setEditingStates((prev) => {
        const current = prev[submissionId];
        if (!current) return prev;
        return {
            ...prev,
            [submissionId]: {
                ...current,
                answers: { ...current.answers, [questionId]: upper as AnswerOption | '-' }
            }
        };
    });
  }

  async function handleSave(submissionId: number) {
    const editing = editingStates[submissionId];
    if (!editing) return;
    
    setError(null);
    setFeedback(null);
    
    const trimmedName = editing.name.trim();
    const trimmedSurname = editing.surname.trim();
    const trimmedEmail = editing.email.trim().toLowerCase();
    const normalizedDni = normalizeDni(editing.dni);

    if (!trimmedName) { setError('El nombre es obligatorio.'); return; }
    if (!trimmedSurname) { setError('Los apellidos son obligatorios.'); return; }
    if (!isValidEmail(trimmedEmail)) { setError('Introduce un email valido.'); return; }
    if (!validateDniNie(normalizedDni)) { setError('Introduce un DNI o NIE valido.'); return; }

    const answersPayload = questions
      .filter((question): question is QuestionEdit & { id: number } => typeof question.id === 'number')
      .map((question) => ({
        question_id: question.id,
        answer: editing.answers[question.id] && editing.answers[question.id] !== '-' ? editing.answers[question.id] : '',
      }));

    if (answersPayload.length !== questions.length) { setError('No se pudieron obtener todas las preguntas del examen.'); return; }

    setSavingIds((prev) => ({ ...prev, [submissionId]: true }));
    try {
      await updateSubmissionAttempt(
        editing.submissionId,
        { name: trimmedName, surname: trimmedSurname, email: trimmedEmail, dni: normalizedDni, answers: answersPayload, merits: editing.merits ?? null },
        token,
      );
      setNeedsStats(true); 
      setRefreshTrigger((prev) => prev + 1);
      
      // Remove from editing states on success
      setEditingStates((prev) => {
        const next = { ...prev };
        delete next[submissionId];
        return next;
      });
      setFeedback('Intento actualizado correctamente.');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSavingIds((prev) => ({ ...prev, [submissionId]: false }));
    }
  }

  async function handleDownloadEmails() {
    setDownloadMessage(null);
    setDownloadError(null);
    setDownloadingEmails(true);
    try {
      const content = await downloadSubmissionEmails(examId, token, {
        search: searchTerm,
        moodleSynced: filterMoodleUsers ? true : undefined,
        type: filterType,
      });
      const lines = content.split(/\r?\n/).map((line) => line.trim()).filter((line) => line !== '');
      const emailCount = lines.length > 0 && lines[0].toLowerCase() === 'email' ? lines.length - 1 : lines.length;
      if (emailCount === 0) { setDownloadError('No hay correos para descargar con los filtros actuales.'); return; }
      const blob = new Blob([lines.join('\n')], { type: 'text/csv;charset=utf-8' });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = `emails_${examId}.csv`;
      document.body.appendChild(anchor);
      anchor.click();
      document.body.removeChild(anchor);
      URL.revokeObjectURL(url);
      setDownloadMessage(`Descargados ${emailCount} emails.`);
    } catch (err) {
      setDownloadError(err instanceof Error ? err.message : String(err));
    } finally {
      setDownloadingEmails(false);
    }
  }

  async function handleDownloadAnalysis() {
    setDownloadMessage(null);
    setDownloadError(null);
    setDownloadingAnalysis(true);
    try {
      const blob = await downloadSubmissionsAnalysis(examId, token, {
        search: searchTerm,
        moodleSynced: filterMoodleUsers ? true : undefined,
        type: filterType,
      });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = `analysis_exam_${examId}.xlsx`;
      document.body.appendChild(anchor);
      anchor.click();
      document.body.removeChild(anchor);
      URL.revokeObjectURL(url);
      setDownloadMessage('Análisis descargado correctamente.');
    } catch (err) {
      setDownloadError(err instanceof Error ? err.message : String(err));
    } finally {
      setDownloadingAnalysis(false);
    }
  }

  const readFileAsBase64 = (file: File): Promise<string> =>
    new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => {
        const result = reader.result as string;
        const commaIndex = result.indexOf(',');
        resolve(commaIndex >= 1 ? result.slice(commaIndex + 1) : result);
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
    setEmailComposeModalError(null);
    setEmailComposeMessage(null);
    setEmailComposerOpen(true);
    setEmailLoading(true);
    setEmailCandidates([]);
    setEmailAttachments([]);
    setEmailSubject(selectedExamName ? `Comunicado sobre ${selectedExamName}` : 'Comunicado oficial');
    setEmailBody('');
    try {
      const emails = await fetchSubmissionEmailList(examId, token, {
        search: searchTerm,
        orderBy,
        orderDir,
        moodleSynced: filterMoodleUsers ? true : undefined,
        type: filterType,
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
    if (!files) return;
    const oversized = Array.from(files).find((f) => f.size > MAX_ATTACHMENT_SIZE_BYTES);
    if (oversized) {
      setEmailComposeModalError(
        `El archivo "${oversized.name}" supera el límite de 5 MB por adjunto.`,
      );
      return;
    }
    setEmailAttachments((prev) => [...prev, ...Array.from(files)]);
  };

  const handleRemoveAttachment = (index: number) => {
    setEmailAttachments((prev) => prev.filter((_, idx) => idx !== index));
  };

  const handleSendEmails = async () => {
    const recipients = emailCandidates.filter((candidate) => candidate.selected).map((item) => item.email);
    if (recipients.length === 0) { setEmailComposeModalError('Selecciona al menos un destinatario.'); return; }
    const trimmedBody = emailBody.trim();
    if (!trimmedBody) { setEmailComposeModalError('El cuerpo del email no puede estar vacío.'); return; }
    setEmailComposeModalError(null);
    setEmailSending(true);
    try {
      const attachments = await Promise.all(
        emailAttachments.map(async (file) => {
          const content = await readFileAsBase64(file);
          return { filename: file.name, content_type: file.type || 'application/octet-stream', content } as SubmissionEmailAttachmentPayload;
        }),
      );
      const payload: SendSubmissionEmailsPayload = {
        exam_id: examId,
        subject: emailSubject || (selectedExamName ? `Comunicado sobre ${selectedExamName}` : 'Comunicado oficial'),
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
    <div className="space-y-6">
       <>
          <SubmissionFilters
            searchTerm={searchTerm}
            onSearchTermChange={handleSearchInput}
            orderBy={orderBy}
            onOrderByChange={handleOrderByChange}
            orderDir={orderDir}
            onOrderDirChange={handleOrderDirChange}
            filterType={filterType}
            onFilterTypeChange={handleFilterTypeChange}
          />

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
            onDownloadAnalysis={handleDownloadAnalysis}
            downloadingAnalysis={downloadingAnalysis}
          />
       </>
       {error && <p className="text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md mb-4">{error}</p>}
       {feedback && <p className="text-green-400 bg-green-400/10 border border-green-500 p-4 rounded-md mb-4">{feedback}</p>}
       
       <SubmissionStats
          totalSubmissions={totalSubmissions}
          averageScore={averageScore}
          averageScoreOfficial={averageScoreOfficial}
          groupAverageScore={groupAverageScore}
          groupAverageScoreOfficial={groupAverageScoreOfficial}
          groupTotalSubmissions={groupTotalSubmissions}
          groupExamNames={groupExamNames}
          needsStats={needsStats}
          selectedExamName={selectedExamName}
          loading={loading}
          hasGroup={(groupExamNames?.length ?? 0) > 1}
          compareGroup={compareGroup}
          onCompareGroupChange={setCompareGroup}
       />
       {loading && <p className="text-brand-yellow">Cargando intentos...</p>}
       
       {!loading && !error && (
         <>
           {totalSubmissions === 0 ? (
             <p className="text-sm text-brand-pink">No hay intentos registrados.</p>
           ) : (
             <>
               <PaginationSettings
                 pageLimit={pageLimit}
                 onLimitChange={handleLimitChange}
                 pageInput={pageInput}
                 onPageInputChange={handlePageInputChange}
                 totalPages={totalPages}
                 goToPage={goToPage}
               />
               
               {submissions.length === 0 && (
                  <p className="text-sm text-brand-blue">Esta pagina no contiene intentos.</p>
               )}

               {submissions.length > 0 && (
                <>
                  <PaginationControls totalItems={totalSubmissions} pageSize={effectiveLimit} currentPage={currentPage} onPageChange={setCurrentPage} />
                  <SubmissionList
                    submissions={submissions}
                    editingStates={editingStates}
                    questions={questions}
                    maxScore={examConfig?.maxScore}
                    secondaryMaxScores={examConfig?.secondaryMaxScores}
                    selectedExamName={selectedExamName}
                    answerOptions={ANSWER_OPTIONS}
                    onStartEditing={startEditing}
                    onCancelEditing={cancelEditing}
                    onDelete={handleDelete}
                    onUpdateField={updateEditingField}
                    onUpdateAnswer={updateAnswer}
                    onSave={handleSave}
                    savingIds={savingIds}
                  />
                  <PaginationControls totalItems={totalSubmissions} pageSize={effectiveLimit} currentPage={currentPage} onPageChange={setCurrentPage} />
                </>
               )}
             </>
           )}
         </>
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
    </div>
  );
}

export default function SubmissionsManager({ exams, token }: SubmissionsManagerProps) {
  const [selectedExamId, setSelectedExamId] = useState<string>('');
  const [syncingMoodle, setSyncingMoodle] = useState(false);
  const [syncMessage, setSyncMessage] = useState<string | null>(null);
  const [syncError, setSyncError] = useState<string | null>(null);

  const handleSyncMoodleUsers = async () => {
    setSyncMessage(null);
    setSyncError(null);
    setSyncingMoodle(true);
    try {
      const result = await syncMoodleUsersStream(token, (msg) => setSyncMessage(msg));
      setSyncMessage(
        `Sincronizados ${result.synced} de ${result.checked} usuarios (fallidos: ${result.failed}).`,
      );
    } catch (err) {
      setSyncError(err instanceof Error ? err.message : String(err));
    } finally {
      setSyncingMoodle(false);
    }
  };

  const selectedExamName = useMemo(() => {
    const numericId = Number(selectedExamId);
    return exams.find((exam) => exam.id === numericId)?.name ?? '';
  }, [exams, selectedExamId]);

  return (
    <div className="mt-8 space-y-6">
      <ExamSelector
        exams={exams}
        selectedExamId={selectedExamId}
        onSelectExam={(value) => setSelectedExamId(value)}
        onSyncMoodleUsers={handleSyncMoodleUsers}
        syncingMoodle={syncingMoodle}
        syncMessage={syncMessage}
        syncError={syncError}
        canSync={!!selectedExamId}
      />

      {!selectedExamId ? (
        <p className="text-sm text-brand-blue">Selecciona un examen para ver los intentos disponibles.</p>
      ) : (
        <SubmissionsViewer 
            key={selectedExamId} 
            examId={Number(selectedExamId)} 
            selectedExamName={selectedExamName}
            token={token} 
        />
      )}
    </div>
  );
}
