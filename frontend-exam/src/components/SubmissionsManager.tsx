import { useCallback, useEffect, useMemo, useState } from 'preact/hooks';

import type { AdminSubmission, AdminSubmissionsResponse, Exam, QuestionEdit } from '../types/exam';
import {
  deleteSubmissionAttempt,
  getExamById,
  getExamSubmissions,
  updateSubmissionAttempt,
} from '../services/adminService';
import { normalizeDni, validateDniNie } from '../utils/validation';

const ANSWER_OPTIONS = ['A', 'B', 'C', 'D'];

type SubmissionStats = {
  totalSubmissions: number;
  averageScore: number | null;
};

const PAGE_SIZE = 25;
const INITIAL_SUBMISSION_STATS: SubmissionStats = {
  totalSubmissions: 0,
  averageScore: null,
};

interface SubmissionsManagerProps {
  exams: Exam[];
  token: string;
}

interface EditingState {
  submissionId: number;
  name: string;
  surname: string;
  email: string;
  dni: string;
  answers: Record<number, string>;
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
  const [currentPage, setCurrentPage] = useState(1);
  const [submissionStats, setSubmissionStats] = useState<SubmissionStats>(() => ({
    ...INITIAL_SUBMISSION_STATS,
  }));
  const [needsStats, setNeedsStats] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [orderBy, setOrderBy] = useState<'submitted_at' | 'score' | 'name' | 'surname'>('submitted_at');
  const [orderDir, setOrderDir] = useState<'asc' | 'desc'>('desc');
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
  }, []);
  const handleSearchInput = (value: string) => {
    setSearchTerm(value);
    setCurrentPage(1);
  };
  const handleOrderByChange = (value: 'submitted_at' | 'score' | 'name' | 'surname') => {
    setOrderBy(value);
    setCurrentPage(1);
  };
  const handleOrderDirChange = (value: 'asc' | 'desc') => {
    setOrderDir(value);
    setCurrentPage(1);
  };

  const selectedExamName = useMemo(() => {
    const numericId = Number(selectedExamId);
    return exams.find((exam) => exam.id === numericId)?.name ?? '';
  }, [exams, selectedExamId]);

  async function loadSubmissionsPage(
    examNumericId: number,
    page: number,
    includeStats = false,
    search = '',
    orderBy: 'submitted_at' | 'score' | 'name' | 'surname' = 'submitted_at',
    orderDir: 'asc' | 'desc' = 'desc',
  ): Promise<AdminSubmissionsResponse | null> {
    const response = await getExamSubmissions(examNumericId, token, {
      limit: PAGE_SIZE,
      offset: (page - 1) * PAGE_SIZE,
      firstLoad: includeStats,
      search,
      orderBy,
      orderDir,
    });
    const knownTotal = includeStats && response.stats_included
      ? response.total_submissions
      : submissionStats.totalSubmissions;
    const totalPages = Math.max(1, Math.ceil(Math.max(knownTotal, 1) / PAGE_SIZE));
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
  const totalPages = Math.max(1, Math.ceil(totalSubmissions / PAGE_SIZE));

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
          needsStats,
          searchTerm,
          orderBy,
          orderDir,
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
  }, [selectedExamId, token, currentPage, searchTerm, orderBy, orderDir]);

  useEffect(() => {
    resetFilters();
  }, [selectedExamId, resetFilters]);

  function handleSelectExam(event: Event) {
    const target = event.currentTarget as HTMLSelectElement;
    setSelectedExamId(target.value);
  }

  function startEditing(submission: AdminSubmission) {
    const initialAnswers: Record<number, string> = {};
    submission.answers.forEach((answer) => {
      initialAnswers[answer.question_id] = answer.answer;
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
    if (!selectedExamId) return;
    if (!confirm('Seguro que quieres eliminar este intento?')) return;

    try {
      await deleteSubmissionAttempt(submissionId, token);
      if (selectedExamId) {
        const examNumericId = Number(selectedExamId);
        if (!Number.isNaN(examNumericId)) {
          await loadSubmissionsPage(examNumericId, currentPage, needsStats, searchTerm, orderBy, orderDir);
        }
      }
      setFeedback('Intento eliminado correctamente.');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

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
    if (!ANSWER_OPTIONS.includes(upper)) return;

    setEditing({
      ...editing,
      answers: {
        ...editing.answers,
        [questionId]: upper,
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
        await loadSubmissionsPage(examNumericId, currentPage, needsStats, searchTerm, orderBy, orderDir);
      }

      setEditing(null);
      setFeedback('Intento actualizado correctamente.');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

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

      <div className="grid gap-4 md:grid-cols-3 mb-4">
        <label className="flex flex-col gap-2">
          <span className="text-xs font-semibold uppercase tracking-[0.35em] text-brand-blue">Buscar</span>
          <input
            type="text"
            placeholder="Nombre, email o DNI"
            value={searchTerm}
            onInput={(event) =>
              handleSearchInput((event.currentTarget as HTMLInputElement).value)
            }
            className="w-full px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-brand-blue focus:border-brand-blue"
          />
        </label>
        <label className="flex flex-col gap-2">
          <span className="text-xs font-semibold uppercase tracking-[0.35em] text-brand-blue">Ordenar por</span>
          <select
            value={orderBy}
            onChange={(event) =>
              handleOrderByChange(
                event.currentTarget.value as 'submitted_at' | 'score' | 'name' | 'surname',
              )
            }
            className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue focus:border-brand-blue"
          >
            <option value="submitted_at">Fecha envío</option>
            <option value="score">Nota</option>
            <option value="name">Nombre</option>
            <option value="surname">Apellido</option>
          </select>
        </label>
        <label className="flex flex-col gap-2">
          <span className="text-xs font-semibold uppercase tracking-[0.35em] text-brand-blue">Dirección</span>
          <select
            value={orderDir}
            onChange={(event) =>
              handleOrderDirChange(event.currentTarget.value as 'asc' | 'desc')
            }
            className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue focus:border-brand-blue"
          >
            <option value="desc">Descendente</option>
            <option value="asc">Ascendente</option>
          </select>
        </label>
      </div>

      {error && (
        <p className="text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md mb-4">{error}</p>
      )}
      {feedback && (
        <p className="text-green-400 bg-green-400/10 border border-green-500 p-4 rounded-md mb-4">{feedback}</p>
      )}

      {selectedExamId && (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 mb-6">
          <div className="rounded-xl border border-brand-pink-soft bg-brand-pink-soft p-5 shadow-lg">
            <p className="text-xs font-semibold uppercase tracking-[0.4em] text-brand-pink">
              Total de intentos
            </p>
            <p className="text-3xl font-extrabold text-brand-pink mt-2">
              {loading ? 'Actualizando...' : totalSubmissions}
            </p>
            {selectedExamName && (
              <p className="text-xs text-brand-yellow mt-2 opacity-90">
                Examen: {selectedExamName}
              </p>
            )}
          </div>
          <div className="rounded-xl border border-brand-blue-soft bg-brand-blue-soft p-5 shadow-lg">
            <p className="text-xs font-semibold uppercase tracking-[0.4em] text-brand-blue">
              Nota media
            </p>
            <p className="text-3xl font-extrabold text-brand-blue mt-2">
              {loading ? 'Actualizando...' : averageScore !== null ? averageScore.toFixed(2) : 'Sin datos'}
            </p>
            {!loading && averageScore === null && (
              <p className="text-xs text-brand-yellow mt-2 opacity-90">Aún no hay notas registradas.</p>
            )}
          </div>
        </div>
      )}

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
        <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between mt-6">
          <button
            type="button"
            disabled={currentPage <= 1}
            onClick={() => setCurrentPage((prev) => Math.max(1, prev - 1))}
            className={`px-4 py-2 rounded border border-brand-blue-soft text-brand-blue transition-colors ${
              currentPage <= 1
                ? 'opacity-40 cursor-not-allowed'
                : 'hover:bg-[#1b2635] hover:border-brand-pink-soft'
            }`}
          >
            Anterior
          </button>
          <p className="text-sm text-gray-300">
            Pagina {currentPage} de {totalPages}
          </p>
          <button
            type="button"
            disabled={currentPage >= totalPages}
            onClick={() => setCurrentPage((prev) => Math.min(totalPages, prev + 1))}
            className={`px-4 py-2 rounded border border-brand-blue-soft text-brand-blue transition-colors ${
              currentPage >= totalPages
                ? 'opacity-40 cursor-not-allowed'
                : 'hover:bg-[#1b2635] hover:border-brand-pink-soft'
            }`}
          >
            Siguiente
          </button>
        </div>
      )}
      {selectedExamId && !loading && submissions.length > 0 && (
        <ul className="list-none p-0 space-y-6">
          {submissions.map((submission) => {
            const user = submission.user;
            const displayName = `${user?.name ?? submission.name ?? ''} ${user?.surname ?? submission.surname ?? ''}`
              .trim()
              || 'Usuario sin nombre';
            const displayEmail = submission.email ?? user?.email ?? 'Sin email registrado';
            const displayDni = submission.dni || user?.dni || 'Sin DNI';
            const marketingAllowed =
              user?.accepts_marketing ??
              submission.accepts_marketing ??
              false;
            const marketingBadgeClass = marketingAllowed
              ? 'bg-brand-yellow-soft text-brand-yellow border border-brand-yellow-soft'
              : 'bg-brand-pink-soft text-brand-pink border border-brand-pink-soft';

            return (
              <li
                key={submission.id}
                className="bg-[#1f2229] border border-brand-blue-soft p-6 rounded-2xl shadow-lg transition-all duration-300 hover:border-brand-pink-soft"
              >
                <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                  <div className="space-y-1">
                    <p className="text-lg font-semibold text-brand-blue">{displayName}</p>
                    <p className="text-sm text-gray-200">
                      <span className="font-semibold text-brand-pink">Email:</span> {displayEmail}
                    </p>
                    <p className="text-sm text-gray-200">
                      <span className="font-semibold text-brand-pink">DNI/NIE:</span> {displayDni}
                    </p>
                    <div className="mt-3 flex flex-wrap gap-2 text-xs font-semibold">
                      <span className="px-3 py-1 rounded-full border border-brand-blue-soft text-brand-blue">
                        Nota: {submission.score ?? 'N/A'}
                      </span>
                      <span className="px-3 py-1 rounded-full border border-brand-yellow-soft text-brand-yellow">
                        Percentil: {submission.percentile ?? 'N/A'}
                      </span>
                    </div>
                    <p className="text-xs text-brand-yellow mt-2 opacity-90">
                      Enviado el {new Date(submission.submitted_at).toLocaleString()}
                    </p>
                    <span
                      className={`mt-3 inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-semibold ${marketingBadgeClass}`}
                    >
                      <span
                        className={`h-2 w-2 rounded-full ${
                          marketingAllowed ? 'bg-brand-yellow' : 'bg-brand-pink'
                        }`}
                      />
                      {marketingAllowed ? 'Acepta comunicaciones de marketing' : 'Marketing no autorizado'}
                    </span>
                  </div>
                  <div className="flex gap-3 mt-4 md:mt-0 flex-wrap">
                    {!editing ? 
                    <button
                      className="py-2 px-4 rounded bg-brand-pink text-dark-200 hover:bg-brand-yellow transition-colors cursor-pointer"
                      onClick={() => startEditing(submission)}
                    >
                      Editar
                    </button>
                    : 
                    <button
                      className="py-2 px-4 rounded border border-brand-pink-soft text-brand-pink hover:bg-brand-pink-soft transition-colors cursor-pointer"
                      onClick={() => setEditing(null)}
                    >
                      Cancelar
                    </button>
                    }
                    <button
                      className="py-2 px-4 rounded bg-red-600 hover:bg-red-700 transition-colors cursor-pointer"
                      onClick={() => handleDelete(submission.id)}
                    >
                      Borrar
                    </button>
                  </div>
                </div>

                {editing && editing.submissionId === submission.id && (
                  <div className="mt-6 border-t border-brand-blue-soft pt-6">
                    <h3 className="text-xl font-semibold mb-4 text-brand-pink">
                      Editar intento ({selectedExamName})
                    </h3>

                    <div className="grid gap-4 md:grid-cols-2">
                      <label className="flex flex-col gap-1">
                        <span className="text-sm font-semibold text-brand-blue">Nombre</span>
                        <input
                          type="text"
                          value={editing.name}
                          onInput={(event) =>
                            updateEditingField(
                              'name',
                              (event.currentTarget as HTMLInputElement).value,
                            )
                          }
                          className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue focus:border-brand-blue"
                        />
                      </label>
                      <label className="flex flex-col gap-1">
                        <span className="text-sm font-semibold text-brand-blue">Apellidos</span>
                        <input
                          type="text"
                          value={editing.surname}
                          onInput={(event) =>
                            updateEditingField(
                              'surname',
                              (event.currentTarget as HTMLInputElement).value,
                            )
                          }
                          className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue focus:border-brand-blue"
                        />
                      </label>
                      <label className="flex flex-col gap-1">
                        <span className="text-sm font-semibold text-brand-blue">Email</span>
                        <input
                          type="email"
                          value={editing.email}
                          onInput={(event) =>
                            updateEditingField(
                              'email',
                              (event.currentTarget as HTMLInputElement).value,
                            )
                          }
                          className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue focus:border-brand-blue"
                        />
                      </label>
                      <label className="flex flex-col gap-1">
                        <span className="text-sm font-semibold text-brand-blue">DNI/NIE</span>
                        <input
                          type="text"
                          value={editing.dni}
                          onInput={(event) =>
                            updateEditingField(
                              'dni',
                              normalizeDni((event.currentTarget as HTMLInputElement).value),
                            )
                          }
                          className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue focus:border-brand-blue"
                        />
                      </label>
                    </div>

                    <div className="mt-6 space-y-4">
                      {questions.map((question, index) => (
                        <div key={question.id ?? index} className="flex flex-wrap items-center gap-3">
                          <div className="flex items-center gap-2">
                            <span className="font-semibold text-brand-pink">Pregunta {index + 1}</span>
                            {question.is_active === false && (
                              <span className="text-xs font-semibold px-2 py-1 rounded bg-amber-500/20 text-amber-300 border border-amber-500/50">
                                Reserva
                              </span>
                            )}
                          </div>
                          <select
                            value={editing.answers[question.id ?? -1] || 'A'}
                            onChange={(event) =>
                              question.id !== undefined &&
                              updateAnswer(question.id, (event.currentTarget as HTMLSelectElement).value)
                            }
                            className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue focus:border-brand-blue"
                          >
                            {ANSWER_OPTIONS.map((option) => (
                              <option key={option} value={option}>
                                {option}
                              </option>
                            ))}
                          </select>
                        </div>
                      ))}
                    </div>

                    <div className="flex flex-wrap gap-3 mt-6">
                      <button
                        className="py-2 px-4 rounded bg-brand-blue text-white font-semibold hover:bg-[#12b2d4] transition-colors cursor-pointer disabled:opacity-60"
                        onClick={handleSave}
                        disabled={saving}
                      >
                        {saving ? 'Guardando...' : 'Guardar cambios'}
                      </button>
                    </div>
                  </div>
                )}
              </li>
            );
          })}
        </ul>
      )}
      {selectedExamId && !loading && totalSubmissions > 0 && (
        <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between mt-6">
          <button
            type="button"
            disabled={currentPage <= 1}
            onClick={() => setCurrentPage((prev) => Math.max(1, prev - 1))}
            className={`px-4 py-2 rounded border border-brand-blue-soft text-brand-blue transition-colors ${
              currentPage <= 1
                ? 'opacity-40 cursor-not-allowed'
                : 'hover:bg-[#1b2635] hover:border-brand-pink-soft'
            }`}
          >
            Anterior
          </button>
          <p className="text-sm text-gray-300">
            Pagina {currentPage} de {totalPages}
          </p>
          <button
            type="button"
            disabled={currentPage >= totalPages}
            onClick={() => setCurrentPage((prev) => Math.min(totalPages, prev + 1))}
            className={`px-4 py-2 rounded border border-brand-blue-soft text-brand-blue transition-colors ${
              currentPage >= totalPages
                ? 'opacity-40 cursor-not-allowed'
                : 'hover:bg-[#1b2635] hover:border-brand-pink-soft'
            }`}
          >
            Siguiente
          </button>
        </div>
      )}
    </section>
  );
}
