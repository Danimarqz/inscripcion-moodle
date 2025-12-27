import type { AdminSubmission, QuestionEdit } from '../../types/exam';
import type { AnswerOption, EditingState } from './types';

interface SubmissionListProps {
  submissions: AdminSubmission[];
  editingStates: Record<number, EditingState>;
  questions: QuestionEdit[];
  selectedExamName: string;
  answerOptions: readonly AnswerOption[];
  onStartEditing: (submission: AdminSubmission) => void;
  onCancelEditing: (submissionId: number) => void;
  onDelete: (submissionId: number) => void;
  onUpdateField: <K extends keyof Omit<EditingState, 'answers' | 'submissionId'>>(
    submissionId: number,
    field: K,
    value: EditingState[K],
  ) => void;
  onUpdateAnswer: (submissionId: number, questionId: number, value: string) => void;
  onSave: (submissionId: number) => Promise<void> | void;
  savingIds: Record<number, boolean>;
}

export default function SubmissionList({
  submissions,
  editingStates,
  questions,
  selectedExamName,
  answerOptions,
  onStartEditing,
  onCancelEditing,
  onDelete,
  onUpdateField,
  onUpdateAnswer,
  onSave,
  savingIds,
}: SubmissionListProps) {
  return (
    <ul className="list-none p-0 space-y-6">
      {submissions.map((submission) => {
        const user = submission.user;
        const displayName = `${user?.name ?? submission.name ?? ''} ${user?.surname ?? submission.surname ?? ''}`
          .trim()
          || 'Usuario sin nombre';
        const displayEmail = submission.email ?? user?.email ?? 'Sin email registrado';
        const displayDni = submission.dni || user?.dni || 'Sin DNI';
        const moodleUserId = user?.moodle_id ?? null;
        const marketingAllowed =
          user?.accepts_marketing ?? submission.accepts_marketing ?? false;
        const marketingBadgeClass = marketingAllowed
          ? 'bg-brand-yellow-soft text-brand-yellow border border-brand-yellow-soft'
          : 'bg-brand-pink-soft text-brand-pink border border-brand-pink-soft';

        const editingState = editingStates[submission.id];
        const isEditing = !!editingState;
        const isSaving = savingIds[submission.id] || false;

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
                <p className="text-xs text-brand-pink-soft mt-1">
                  Tipo: <span className="text-white">{submission.selected_result_type || 'General'}</span>
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
                <div className="mt-3 flex flex-wrap gap-2 text-xs">
                  {moodleUserId ? (
                    <a
                      href={`https://moodle.opositatcae.es/user/view.php?id=${moodleUserId}`}
                      target="_blank"
                      rel="noreferrer"
                      className="inline-flex items-center gap-1 font-semibold text-brand-blue hover:underline"
                    >
                      Ver en moodle
                    </a>
                  ) : (
                    <span className="text-gray-400">Sin usuario Moodle asignado</span>
                  )}
                </div>
              </div>
              <div className="flex gap-3 mt-4 md:mt-0 flex-wrap">
                {!isEditing ? (
                  <button
                    className="py-2 px-4 rounded bg-brand-pink text-dark-200 hover:bg-brand-yellow transition-colors cursor-pointer"
                    onClick={() => onStartEditing(submission)}
                  >
                    Editar
                  </button>
                ) : (
                  <button
                    className="py-2 px-4 rounded border border-brand-pink-soft text-brand-pink hover:bg-brand-pink-soft transition-colors cursor-pointer"
                    onClick={() => onCancelEditing(submission.id)}
                  >
                    Cancelar
                  </button>
                )}
                <button
                  className="py-2 px-4 rounded bg-red-600 hover:bg-red-700 transition-colors cursor-pointer"
                  onClick={() => onDelete(submission.id)}
                >
                  Borrar
                </button>
              </div>
            </div>
            {isEditing && editingState && (
              <div className="mt-6 border-t border-brand-blue-soft pt-6">
                <h3 className="text-xl font-semibold mb-4 text-brand-pink">
                  Editar intento ({selectedExamName})
                </h3>
                <div className="grid gap-4 md:grid-cols-2">
                  <label className="flex flex-col gap-1">
                    <span className="text-sm font-semibold text-brand-blue">Nombre</span>
                    <input
                      type="text"
                      value={editingState.name}
                      onInput={(event) =>
                        onUpdateField(
                          submission.id,
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
                      value={editingState.surname}
                      onInput={(event) =>
                        onUpdateField(
                          submission.id,
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
                      value={editingState.email}
                      onInput={(event) =>
                        onUpdateField(
                          submission.id,
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
                      value={editingState.dni}
                      onInput={(event) =>
                        onUpdateField(
                          submission.id,
                          'dni',
                          (event.currentTarget as HTMLInputElement).value,
                        )
                      }
                      className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue focus:border-brand-blue"
                    />
                  </label>
                </div>
                <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
                  {questions.map((question, index) => (
                    <div
                      key={question.id ?? index}
                      className="flex flex-col gap-4 rounded border border-[#444] bg-[#171a23] p-4"
                    >
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-brand-pink">Pregunta {index + 1}</span>
                        {question.is_active === false && (
                          <span className="text-xs font-semibold px-2 py-1 rounded bg-amber-500/20 text-amber-300 border border-amber-500/50">
                            Reserva
                          </span>
                        )}
                      </div>
                      <select
                        value={editingState.answers[question.id ?? -1] || 'A'}
                        onChange={(event) =>
                          question.id !== undefined &&
                          onUpdateAnswer(
                            submission.id,
                            question.id,
                            (event.currentTarget as HTMLSelectElement).value as AnswerOption,
                          )
                        }
                        className="w-full rounded border border-[#444] bg-[#1f2229] px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-brand-blue focus:border-brand-blue"
                      >
                        {answerOptions.map((option) => (
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
                    onClick={() => onSave(submission.id)}
                    disabled={isSaving}
                  >
                    {isSaving ? 'Guardando...' : 'Guardar cambios'}
                  </button>
                </div>
              </div>
            )}
          </li>
        );
      })}
    </ul>
  );
}
