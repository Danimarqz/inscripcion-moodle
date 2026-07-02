import type { SubmissionBreakdown } from '../../types/exam';

interface SubmissionBreakdownChipsProps {
  breakdown?: SubmissionBreakdown | null;
}

// SubmissionBreakdownChips is the compact per-row summary shown on each list
// card: one small chip per group (name + score/max, colored by eliminatory
// pass) plus aciertos / fallos / en blanco. Sits alongside the score/percentile
// badges; the full detail lives in the edit view (SubmissionBreakdownCard).
export default function SubmissionBreakdownChips({ breakdown }: SubmissionBreakdownChipsProps) {
  if (!breakdown) return null;
  const groups = Array.isArray(breakdown.groups) ? breakdown.groups : [];
  return (
    <div className="mt-2 flex flex-wrap gap-2 text-xs font-semibold">
      {groups.map((g) => {
        const cls = g.eliminatory
          ? g.passed
            ? 'border-green-500/50 text-green-300'
            : 'border-red-500/50 text-red-300'
          : 'border-brand-blue-soft text-brand-blue/80';
        return (
          <span key={g.group_id} className={`px-3 py-1 rounded-full border ${cls}`}>
            {g.name}: {g.score}/{g.max_score}
          </span>
        );
      })}
      <span className="px-3 py-1 rounded-full border border-teal-500/40 text-teal-200">
        Aciertos: {breakdown.correct_answers}
      </span>
      <span className="px-3 py-1 rounded-full border border-brand-pink-soft text-brand-pink">
        Fallos: {breakdown.incorrect_answers}
      </span>
      <span className="px-3 py-1 rounded-full border border-gray-500/40 text-gray-300">
        En blanco: {breakdown.not_answered}
      </span>
    </div>
  );
}
