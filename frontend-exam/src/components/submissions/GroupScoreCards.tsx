import type { GroupScore } from '../../types/exam';

interface GroupScoreCardsProps {
  groups?: GroupScore[] | null;
}

// GroupScoreCards renders the per-group result cards (name, score/max,
// eliminatory pass badge, minimum threshold). Extracted from SubmissionSummary
// so the student view and the admin submission detail share one render path
// instead of duplicating the groups JSX.
export default function GroupScoreCards({ groups }: GroupScoreCardsProps) {
  if (!Array.isArray(groups) || groups.length === 0) return null;
  return (
    <div className="mt-6">
      <h3 className="text-base font-semibold text-brand-yellow mb-3">Resultado por grupo</h3>
      <div className="flex flex-wrap gap-4">
        {groups.map((g) => (
          <div
            key={g.group_id}
            className="flex-1 min-w-[200px] rounded-xl bg-[#1f2a24] border border-green-500/30 p-4"
          >
            <div className="flex items-center justify-between gap-2">
              <p className="text-sm font-semibold text-green-100">{g.name}</p>
              {g.eliminatory && (
                <span
                  className={`text-xs font-semibold rounded-full px-2 py-0.5 border ${
                    g.passed
                      ? 'border-green-500/50 text-green-300 bg-green-500/20'
                      : 'border-red-500/50 text-red-300 bg-red-500/20'
                  }`}
                >
                  {g.passed ? 'Superado' : 'No superado'}
                </span>
              )}
            </div>
            <p className="text-2xl font-bold text-green-200 mt-1">
              {g.score} <span className="text-base text-green-200/60">/ {g.max_score}</span>
            </p>
            {typeof g.min_passing_score === 'number' && (
              <p className="text-xs text-green-200/50 mt-1">Mínimo: {g.min_passing_score}</p>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
