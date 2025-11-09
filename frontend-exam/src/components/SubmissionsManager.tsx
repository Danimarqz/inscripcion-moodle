import { useEffect, useMemo, useState } from 'preact/hooks';

import type { AdminSubmission, Exam, QuestionEdit } from '../types/exam';
import {
  deleteSubmissionAttempt,
  getExamById,
  getExamSubmissions,
  updateSubmissionAttempt,
} from '../services/adminService';
import { normalizeDni, validateDniNie } from '../utils/validation';

const ANSWER_OPTIONS = ['A', 'B', 'C', 'D'];

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

  const selectedExamName = useMemo(() => {
    const numericId = Number(selectedExamId);
    return exams.find((exam) => exam.id === numericId)?.name ?? '';
  }, [exams, selectedExamId]);

  const { totalAttempts, scoredAttempts, averageScore } = useMemo(() => {
    const total = submissions.length;
    if (total === 0) {
      return { totalAttempts: 0, scoredAttempts: 0, averageScore: null };
    }

    const scores = submissions
      .map((submission) => submission.score)
      .filter((score): score is number => typeof score === 'number' && Number.isFinite(score));

    if (scores.length === 0) {
      return { totalAttempts: total, scoredAttempts: 0, averageScore: null };
    }

    const average = scores.reduce((sum, value) => sum + value, 0) / scores.length;
    return { totalAttempts: total, scoredAttempts: scores.length, averageScore: average };
  }, [submissions]);

  useEffect(() => {
    async function loadData(examNumericId: number) {
      setLoading(true);
      setError(null);
      setFeedback(null);
      setEditing(null);
      try {
        const [submissionData, examData] = await Promise.all([
          getExamSubmissions(examNumericId, token),
          getExamById(examNumericId, token),
        ]);
        setSubmissions(submissionData);
        setQuestions(examData.questions ?? []);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
        setSubmissions([]);
        setQuestions([]);
      } finally {
        setLoading(false);
      }
    }

    if (!selectedExamId) {
      setSubmissions([]);
      setQuestions([]);
      setEditing(null);
      setFeedback(null);
      return;
    }

    const examNumericId = Number(selectedExamId);
    if (Number.isNaN(examNumericId)) {
      setError('Identificador de examen invalido.');
      setSubmissions([]);
      setQuestions([]);
      return;
    }

    loadData(examNumericId);
  }, [selectedExamId, token]);

  function handleSelectExam(event: Event) {
    const value = (event.target as HTMLSelectElement).value;
    setSelectedExamId(value);
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
      setSubmissions((prev) => prev.filter((item) => item.id !== submissionId));
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

      setSubmissions((prev) => prev.map((item) => (item.id === updated.id ? updated : item)));

      const examNumericId = Number(selectedExamId);
      if (selectedExamId && !Number.isNaN(examNumericId)) {
        const refreshedSubmissions = await getExamSubmissions(examNumericId, token);
        setSubmissions(refreshedSubmissions);
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
    <section className="mt-8">

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

      {selectedExamId && (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 mb-6">
          <div className="rounded-lg border border-[#3a3f49] bg-[#1f2229] p-4">
            <p className="text-sm text-gray-400">Total de intentos</p>
            <p className="text-2xl font-bold text-white">
              {loading ? 'Actualizando...' : totalAttempts}
            </p>
            {selectedExamName && (
              <p className="text-xs text-gray-500 mt-1">Examen: {selectedExamName}</p>
            )}
          </div>
          <div className="rounded-lg border border-[#3a3f49] bg-[#1f2229] p-4">
            <p className="text-sm text-gray-400">Nota media</p>
            <p className="text-2xl font-bold text-white">
              {loading ? 'Actualizando...' : averageScore !== null ? averageScore.toFixed(2) : 'Sin datos'}
            </p>
            {!loading && scoredAttempts > 0 && (
              <p className="text-xs text-gray-500 mt-1">
                Calculada sobre {scoredAttempts}{' '}
                {scoredAttempts === 1 ? 'intento con nota' : 'intentos con nota'}
              </p>
            )}
            {!loading && scoredAttempts === 0 && (
              <p className="text-xs text-gray-500 mt-1">Aún no hay notas registradas.</p>
            )}
          </div>
        </div>
      )}

      {!selectedExamId && <p>Selecciona un examen para ver los intentos disponibles.</p>}

      {selectedExamId && loading && <p>Cargando intentos...</p>}

      {selectedExamId && !loading && submissions.length === 0 && !error && (
        <p>No hay intentos para este examen.</p>
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
              ? 'bg-green-500/15 text-green-300 border border-green-500/40'
              : 'bg-red-500/15 text-red-300 border border-red-500/40';

            return (
              <li key={submission.id} className="bg-[#2a2d33] p-6 rounded-lg shadow-lg">
                <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                  <div>
                    <p className="font-semibold">{displayName}</p>
                    <p className="text-sm text-gray-400">Email: {displayEmail}</p>
                    <p className="text-sm text-gray-400">DNI/NIE: {displayDni}</p>
                    <p className="text-sm text-gray-400">
                      Puntuacion: {submission.score ?? 'N/A'} | Percentil: {submission.percentile ?? 'N/A'}
                    </p>
                    <p className="text-xs text-gray-500">Enviado el {new Date(submission.submitted_at).toLocaleString()}</p>
                    <span className={`mt-2 inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-semibold ${marketingBadgeClass}`}>
                      <span
                        className={`h-2 w-2 rounded-full ${
                          marketingAllowed ? 'bg-green-300' : 'bg-red-300'
                        }`}
                      />
                      {marketingAllowed ? 'Acepta comunicaciones de marketing' : 'Marketing no autorizado'}
                    </span>
                  </div>
                  <div className="flex gap-3 mt-4 md:mt-0">
                    {!editing ? 
                    <button
                      className="py-2 px-4 rounded bg-purple-600 hover:bg-purple-700 transition-colors cursor-pointer"
                      onClick={() => startEditing(submission)}
                    >
                      Editar
                    </button>
                    : 
                    <button
                      className="py-2 px-4 rounded bg-gray-600 hover:bg-gray-700 transition-colors cursor-pointer"
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
                  <div className="mt-6 border-t border-[#444] pt-6">
                    <h3 className="text-xl font-semibold mb-4">
                      Editar intento ({selectedExamName})
                    </h3>

                    <div className="grid gap-4 md:grid-cols-2">
                      <label className="flex flex-col gap-1">
                        <span>Nombre</span>
                        <input
                          type="text"
                          value={editing.name}
                          onInput={(event) =>
                            updateEditingField('name', (event.target as HTMLInputElement).value)
                          }
                          className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white"
                        />
                      </label>
                      <label className="flex flex-col gap-1">
                        <span>Apellidos</span>
                        <input
                          type="text"
                          value={editing.surname}
                          onInput={(event) =>
                            updateEditingField('surname', (event.target as HTMLInputElement).value)
                          }
                          className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white"
                        />
                      </label>
                      <label className="flex flex-col gap-1">
                        <span>Email</span>
                        <input
                          type="email"
                          value={editing.email}
                          onInput={(event) =>
                            updateEditingField('email', (event.target as HTMLInputElement).value)
                          }
                          className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white"
                        />
                      </label>
                      <label className="flex flex-col gap-1">
                        <span>DNI/NIE</span>
                        <input
                          type="text"
                          value={editing.dni}
                          onInput={(event) =>
                            updateEditingField('dni', normalizeDni((event.target as HTMLInputElement).value))
                          }
                          className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white"
                        />
                      </label>
                    </div>

                    <div className="mt-6 space-y-4">
                      {questions.map((question, index) => (
                        <div key={question.id ?? index} className="flex flex-wrap items-center gap-3">
                          <div className="flex items-center gap-2">
                            <span className="font-semibold">Pregunta {index + 1}</span>
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
                              updateAnswer(question.id, (event.target as HTMLSelectElement).value)
                            }
                            className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white"
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

                    <div className="flex gap-3 mt-6">
                      <button
                        className="py-2 px-4 rounded bg-green-600 hover:bg-green-700 transition-colors cursor-pointer disabled:opacity-60"
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
    </section>
  );
}
