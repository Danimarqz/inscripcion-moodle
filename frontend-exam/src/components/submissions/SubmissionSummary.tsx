import type { AnswerReview } from '../../types/exam';

interface SubmissionSummaryProps {
  message: string;
  review: AnswerReview[] | null;
  showScore: boolean;
  showScoreFull: boolean;
  showPercentile: boolean;
  score: number | null;
  correctAnswers: number | null;
  totalQuestions: number | null;
  percentile: number | null;
  position: number | null;
  totalSubmissions: number | null;
}

export default function SubmissionSummary({
  message,
  review,
  showScore,
  showScoreFull,
  showPercentile,
  score,
  correctAnswers,
  totalQuestions,
  percentile,
  position,
  totalSubmissions,
}: SubmissionSummaryProps) {
  const trimmedMessage = message.trim();
  if (!trimmedMessage) return null;

  const sentenceMatch = trimmedMessage.match(/.*?[.!?](?:\s|$)/);
  const primaryMessage = (sentenceMatch ? sentenceMatch[0] : trimmedMessage).trim();

  return (
    <div className="mt-6 rounded-2xl border border-green-500/60 bg-green-500/5 p-6 shadow-[0_10px_25px_rgba(16,185,129,0.15)] text-left">
      <p className="text-green-200 text-lg font-semibold mb-2">
        {primaryMessage ?? trimmedMessage}
      </p>

      <div className="flex flex-wrap gap-4">
        {showScore && typeof score === 'number' && (
          <div className="flex-1 min-w-[180px] rounded-xl bg-[#1f2a24] border border-green-500/30 p-4">
            <p className="text-xs uppercase tracking-widest text-green-400/80">Puntuaci�n</p>
            <p className="text-2xl font-bold text-green-200">{score}</p>
          </div>
        )}
        {showScoreFull && typeof correctAnswers === 'number' && typeof totalQuestions === 'number' && (
          <div className="flex-1 min-w-[180px] rounded-xl bg-[#1f252a] border border-teal-500/30 p-4">
            <p className="text-xs uppercase tracking-widest text-teal-300/80">Aciertos</p>
            <p className="text-2xl font-bold text-teal-200">
              {correctAnswers} <span className="text-base text-teal-200/70">de</span> {totalQuestions}
            </p>
          </div>
        )}
        {showPercentile && typeof percentile === 'number' && (
          <div className="flex-1 min-w-[180px] rounded-xl bg-[#1f2330] border border-indigo-500/30 p-4">
            <p className="text-xs uppercase tracking-widest text-indigo-300/80">Percentil</p>
            <p className="text-2xl font-bold text-indigo-200">{percentile}</p>
            {typeof position === 'number' && typeof totalSubmissions === 'number' && (
              <p className="text-sm text-indigo-200/70 mt-1">
                Posici�n {position} de {totalSubmissions}
              </p>
            )}
          </div>
        )}
      </div>
      {Array.isArray(review) && review.length > 0 && (
        <div className="mt-6">
          <h3 className="text-base font-semibold text-brand-yellow mb-3">
            Detalle de respuestas
          </h3>
          <div className="overflow-x-auto rounded-2xl border border-brand-blue-soft bg-[#12141b]">
            <table className="min-w-full text-sm">
              <thead className="text-xs uppercase tracking-[0.35em] text-brand-yellow/80">
                <tr>
                  <th className="px-4 py-2 text-left">Pregunta</th>
                  <th className="px-4 py-2 text-left">Tu respuesta</th>
                  <th className="px-4 py-2 text-left">Respuesta correcta</th>
                  <th className="px-4 py-2 text-left">Estado</th>
                </tr>
              </thead>
              <tbody>
                {review.map((item, index) => {
                  const questionNumber = item.question_label ?? index + 1;
                  const selected = item.selected_option ?? 'Sin marcar';
                  const correct = item.correct_option ?? '-';
                  const isAnswered = Boolean(item.selected_option);
                  const statusLabel = item.is_correct
                    ? 'Correcta'
                    : isAnswered
                      ? 'Incorrecta'
                      : 'Sin responder';
                  const rowClass = item.is_correct
                    ? 'bg-green-500/5'
                    : isAnswered
                      ? 'bg-brand-pink-soft'
                      : '';
                  const statusClass = item.is_correct
                    ? 'text-green-300'
                    : isAnswered
                      ? 'text-brand-pink'
                      : 'text-gray-400';
                  return (
                    <tr key={`${item.question_id}-${index}`} className={rowClass}>
                      <td className="px-4 py-2 font-semibold text-white">
                        Pregunta {questionNumber}
                      </td>
                      <td className="px-4 py-2">
                        <span
                          className={`inline-flex items-center rounded-full border px-3 py-1 text-xs font-semibold ${
                            item.is_correct
                              ? 'border-green-500/40 text-green-300'
                              : isAnswered
                                ? 'border-brand-pink-soft text-brand-pink'
                                : 'border-gray-600 text-gray-400'
                          }`}
                        >
                          {selected}
                        </span>
                      </td>
                      <td className="px-4 py-2 text-brand-yellow font-semibold">{correct}</td>
                      <td className={`px-4 py-2 text-sm font-semibold ${statusClass}`}>{statusLabel}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
