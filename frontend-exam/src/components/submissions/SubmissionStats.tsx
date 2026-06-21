interface SubmissionStatsProps {
  totalSubmissions: number;
  averageScore: number | null;
  averageScoreOfficial: number | null;
  groupTotalSubmissions: number | null;
  groupExamNames: string[] | null;
  needsStats: boolean;
  selectedExamName: string;
  loading: boolean;
  hasGroup: boolean;
  compareGroup: boolean;
  onCompareGroupChange: (value: boolean) => void;
}

export default function SubmissionStats({
  totalSubmissions,
  averageScore,
  averageScoreOfficial,
  groupTotalSubmissions,
  groupExamNames,
  selectedExamName,
  loading,
  hasGroup,
  compareGroup,
  onCompareGroupChange,
}: SubmissionStatsProps) {
  const showGroup = compareGroup && groupExamNames && groupExamNames.length > 1;
  const displayTotal = showGroup && groupTotalSubmissions != null ? groupTotalSubmissions : totalSubmissions;
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 mb-6">
      <div className="rounded-xl border border-brand-pink-soft bg-brand-pink-soft p-5 shadow-lg">
        <p className="text-xs font-semibold uppercase tracking-[0.4em] text-brand-pink">Total de intentos</p>
        <p className="text-3xl font-extrabold text-brand-pink mt-2">
          {displayTotal}
        </p>
        {showGroup ? (
          <p className="text-xs text-brand-yellow mt-2 opacity-90">Exámenes: {groupExamNames!.join(', ')}</p>
        ) : (
          selectedExamName && (
            <p className="text-xs text-brand-yellow mt-2 opacity-90">Examen: {selectedExamName}</p>
          )
        )}
      </div>
      <div className="rounded-xl border border-brand-blue-soft bg-brand-blue-soft p-5 shadow-lg">
        <p className="text-xs font-semibold uppercase tracking-[0.4em] text-brand-blue">
          Nota media{compareGroup ? ' (grupo)' : ''}
        </p>
        <div className="flex items-baseline gap-4 mt-2">
          <span className="text-3xl font-extrabold text-brand-blue">
            {averageScore !== null ? averageScore.toFixed(2) : 'Sin datos'}
          </span>
          {averageScoreOfficial !== null && (
            <span className="text-sm font-semibold text-brand-blue/80">
              Oficial: {averageScoreOfficial.toFixed(2)}
            </span>
          )}
        </div>
        {!loading && averageScore === null && (
          <p className="text-xs text-brand-yellow mt-2 opacity-90">Aún no hay notas registradas.</p>
        )}
        {hasGroup && (
          <label className="flex items-center gap-2 mt-3 cursor-pointer text-xs text-brand-blue">
            <input
              type="checkbox"
              checked={compareGroup}
              onChange={(e) => onCompareGroupChange((e.currentTarget as HTMLInputElement).checked)}
              className="h-3.5 w-3.5 accent-brand-blue cursor-pointer"
            />
            Comparar media contra el grupo de exámenes asociados
          </label>
        )}
      </div>
    </div>
  );
}
