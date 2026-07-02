import type { SubmissionBreakdown } from '../../types/exam';
import GroupScoreCards from './GroupScoreCards';

interface SubmissionBreakdownCardProps {
  breakdown: SubmissionBreakdown;
}

// SubmissionBreakdownCard renders the admin-facing score breakdown for a single
// submission: global pass status, correct/blank/wrong counts, and the per-group
// cards (reusing GroupScoreCards, shared with the student view). General score
// and percentile stay on the submission header badges, so they are not repeated
// here. Mirrors the visual style of the student's SubmissionSummary block.
export default function SubmissionBreakdownCard({ breakdown }: SubmissionBreakdownCardProps) {
  const {
    correct_answers,
    incorrect_answers,
    not_answered,
    total_questions,
    is_passed,
    groups,
  } = breakdown;

  return (
    <div className="mt-6 rounded-2xl border border-green-500/60 bg-green-500/5 p-6 text-left">
      <div className="flex items-center gap-3 mb-4">
        <p className="text-green-200 text-lg font-semibold">Desglose del intento</p>
        {is_passed === true && (
          <span className="inline-flex items-center rounded-full bg-green-500/20 border border-green-500/50 px-3 py-1 text-xs font-semibold text-green-300">
            Aprobado
          </span>
        )}
        {is_passed === false && (
          <span className="inline-flex items-center rounded-full bg-gray-500/20 border border-gray-500/50 px-3 py-1 text-xs font-semibold text-gray-400">
            No aprobado
          </span>
        )}
      </div>
      <div className="flex flex-wrap gap-4">
        <div className="flex-1 min-w-[180px] rounded-xl bg-[#1f252a] border border-teal-500/30 p-4">
          <p className="text-xs uppercase tracking-widest text-teal-300/80">Aciertos</p>
          <p className="text-2xl font-bold text-teal-200">
            {correct_answers} <span className="text-base text-teal-200/70">de</span> {total_questions}
          </p>
          <div className="mt-2 text-xs flex gap-3 text-teal-200/50">
            <span>
              Fallos <span className="text-brand-pink-soft font-semibold">{incorrect_answers}</span>
            </span>
            <span>
              En Blanco <span className="text-gray-400 font-semibold">{not_answered}</span>
            </span>
          </div>
        </div>
      </div>
      <GroupScoreCards groups={groups} />
    </div>
  );
}
