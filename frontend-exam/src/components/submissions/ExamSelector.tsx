import type { Exam } from '../../types/exam';

interface ExamSelectorProps {
  exams: Exam[];
  selectedExamId: string;
  onSelectExam: (value: string) => void;
  onSyncMoodleUsers: () => void;
  syncingMoodle: boolean;
  syncMessage: string | null;
  syncError: string | null;
  canSync: boolean;
}

export default function ExamSelector({
  exams,
  selectedExamId,
  onSelectExam,
  onSyncMoodleUsers,
  syncingMoodle,
  syncMessage,
  syncError,
  canSync,
}: ExamSelectorProps) {
  return (
    <div className="flex flex-wrap items-end gap-3 mb-4">
      <label className="flex-1 min-w-[240px]">
        <span className="block font-semibold mb-2 text-brand-pink tracking-[0.35em] uppercase text-xs">
          Selecciona un examen
        </span>
        <select
          value={selectedExamId}
          onChange={(event) => onSelectExam(event.currentTarget.value)}
          className="w-full px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white transition-colors focus:outline-none focus:ring-2 focus:ring-brand-blue focus:border-brand-blue"
        >
          <option value="">-- Escoge un examen --</option>
          {exams.map((exam) => (
            <option key={exam.id} value={exam.id}>
              {exam.name} (ID: {exam.id})
            </option>
          ))}
        </select>
      </label>
      {canSync && (
        <button
          type="button"
          onClick={onSyncMoodleUsers}
          disabled={syncingMoodle}
          className={`px-4 py-2 rounded border border-brand-blue-soft text-white transition-colors ${
            syncingMoodle
              ? 'bg-[#1c2230] opacity-60 cursor-progress'
              : 'bg-brand-blue hover:bg-[#12b2d4] cursor-pointer'
          }`}
        >
          {syncingMoodle ? 'Sincronizando usuarios Moodle...' : 'Sincronizar usuarios Moodle'}
        </button>
      )}
      {syncMessage && <p className="text-xs text-brand-blue w-full md:w-auto">{syncMessage}</p>}
      {syncError && <p className="text-xs text-brand-pink w-full md:w-auto">{syncError}</p>}
    </div>
  );
}
