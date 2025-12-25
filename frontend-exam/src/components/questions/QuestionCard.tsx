import { memo } from 'preact/compat';
import { ANSWER_OPTIONS } from '../../constants/answerOptions';
import type { Question } from '../../types/exam';
import QuestionCardFrame from './QuestionCardFrame';

interface QuestionCardProps {
  question: Question;
  position: number;
  isReserve: boolean;
  selectedOption: string | null;
  onSetAnswer: (questionId: number, option: string) => void;
  onClearAnswer: (questionId: number) => void;
}

function QuestionCard({
  question,
  position,
  isReserve,
  selectedOption,
  onSetAnswer,
  onClearAnswer,
}: QuestionCardProps) {
  const questionId = question.id;
  if (typeof questionId !== 'number') {
    return null;
  }
  const isCancelled = question.is_cancelled === true;
  const displayName = question.name ?? position;
  const label = `${isReserve ? 'Reserva' : 'Pregunta'} ${displayName}`;
  const badgeConfig = isCancelled
    ? {
        text: 'Anulada',
        className: 'bg-red-500/20 text-red-300 border border-red-500/50',
      }
    : {
        text: isReserve ? 'Reserva' : 'Activa',
        className: isReserve
          ? 'bg-amber-500/20 text-amber-300 border border-amber-500/50'
          : 'bg-green-500/20 text-green-300 border border-green-500/50',
      };

  const meta = isCancelled ? (
    <p className="text-sm text-red-300 mb-4">
      Esta pregunta se ha marcado como anulada y no cuenta para la nota final.
    </p>
  ) : null;

  return (
    <QuestionCardFrame
      label={label}
      badgeText={badgeConfig.text}
      badgeClassName={badgeConfig.className}
      meta={meta}
    >
      <ul className="list-none p-0 m-0 flex flex-wrap gap-3">
        {ANSWER_OPTIONS.map((optionChar) => {
          const isSelected = selectedOption === optionChar;
          return (
            <li key={`${questionId}-${optionChar}`}>
              <button
                type="button"
                onClick={() => {
                  if (isCancelled) return;
                  if (isSelected) {
                    onClearAnswer(questionId);
                  } else {
                    onSetAnswer(questionId, optionChar);
                  }
                }}
                disabled={isCancelled}
                aria-pressed={isSelected}
                className={`w-16 text-center px-4 py-2 text-lg font-semibold rounded-md border transition-all duration-200 ${
                  isCancelled
                    ? 'border-gray-600 text-gray-600 cursor-not-allowed'
                    : isSelected
                    ? 'border-brand-yellow bg-brand-yellow text-dark-200 shadow-lg cursor-pointer'
                    : 'border-[#555] text-gray-200 hover:border-brand-blue hover:text-brand-yellow cursor-pointer'
                }`}
              >
                {optionChar}
              </button>
            </li>
          );
        })}
      </ul>
      <div className="mt-4 flex flex-wrap items-center gap-3">
        <p className="text-xs text-gray-500">
          Usa los botones para marcar tu opción; si pulsas de nuevo o en &ldquo;Borrar respuesta&rdquo; la
          pregunta queda en blanco.
        </p>
        <button
          type="button"
          onClick={() => onClearAnswer(questionId)}
          disabled={!selectedOption}
          className="text-xs font-semibold px-3 py-1 rounded border border-brand-blue-soft text-brand-blue hover:bg-brand-blue-soft disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
        >
          Borrar respuesta
        </button>
      </div>
    </QuestionCardFrame>
  );
}

export default memo(QuestionCard);
