import type { Question } from '../../types/exam';
import QuestionCard from './QuestionCard';
import QuestionSection from './QuestionSection';

type QuestionEntry = { index: number; question: Question };

interface QuestionListProps {
  activeEntries: QuestionEntry[];
  reserveEntries: QuestionEntry[];
  userAnswers: Record<number, string>;
  onSetAnswer: (questionId: number, option: string) => void;
  onClearAnswer: (questionId: number) => void;
}

export default function QuestionList({
  activeEntries,
  reserveEntries,
  userAnswers,
  onSetAnswer,
  onClearAnswer,
}: QuestionListProps) {
  return (
    <>
      <QuestionSection
        title="Preguntas activas"
        entries={activeEntries}
        description={
          <p className="text-sm text-gray-400 mt-1">
            Responde todas las preguntas activas; puntúan en la nota final.
          </p>
        }
        renderEntry={(entry, index) => {
          const { question, index: originalIndex } = entry;
          const mapKey = `${question.id}-active-${originalIndex}`;
          return (
            <QuestionCard
              key={mapKey}
              question={question}
              position={index + 1}
              isReserve={false}
              selectedOption={userAnswers[question.id] ?? null}
              onSetAnswer={onSetAnswer}
              onClearAnswer={onClearAnswer}
            />
          );
        }}
        showCount={false}
        titleTag="h2"
        titleClassName="text-2xl font-semibold text-brand-pink"
        sectionClassName="mt-8"
      />

      {reserveEntries.length > 0 && (
        <QuestionSection
          title="Preguntas de reserva"
          entries={reserveEntries}
          renderEntry={(entry, index) => {
            const { question, index: originalIndex } = entry;
            const mapKey = `${question.id}-reserve-${originalIndex}`;
            return (
              <QuestionCard
                key={mapKey}
                question={question}
                position={activeEntries.length + index + 1}
                isReserve={true}
                selectedOption={userAnswers[question.id] ?? null}
                onSetAnswer={onSetAnswer}
                onClearAnswer={onClearAnswer}
              />
            );
          }}
          showCount={false}
          titleTag="h2"
          titleClassName="text-2xl font-semibold text-brand-pink"
          sectionClassName="mt-8"
        />
      )}
    </>
  );
}

