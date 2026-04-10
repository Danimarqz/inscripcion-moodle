import type { Question } from '../../types/exam';
import QuestionCard from './QuestionCard';

type QuestionEntry = { index: number; question: Question };

interface QuestionListProps {
  entries: QuestionEntry[];
  userAnswers: Record<number, string>;
  onSetAnswer: (questionId: number, option: string) => void;
  onClearAnswer: (questionId: number) => void;
}

export default function QuestionList({
  entries,
  userAnswers,
  onSetAnswer,
  onClearAnswer,
}: QuestionListProps) {
  return (
    <section className="mt-8">
      <h2 className="text-2xl font-semibold text-brand-pink">Preguntas</h2>
      <p className="text-sm text-gray-400 mt-1 mb-4">
        Responde todas las preguntas. Las de reserva aparecen intercaladas según su número y
        solo puntúan si sustituyen a una pregunta activa invalidada.
      </p>
      {entries.map((entry) => {
        const { question, index: originalIndex } = entry;
        const isReserve = question.is_active === false;
        const mapKey = `${question.id}-${originalIndex}`;
        return (
          <QuestionCard
            key={mapKey}
            question={question}
            position={question.name ?? originalIndex + 1}
            isReserve={isReserve}
            selectedOption={userAnswers[question.id] ?? null}
            onSetAnswer={onSetAnswer}
            onClearAnswer={onClearAnswer}
          />
        );
      })}
    </section>
  );
}
