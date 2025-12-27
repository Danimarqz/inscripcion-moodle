interface SubmissionStatsProps {
  totalSubmissions: number;
  averageScore: number | null;
  needsStats: boolean;
  selectedExamName: string;
  loading: boolean;
}

export default function SubmissionStats({
  totalSubmissions,
  averageScore,
  needsStats,
  selectedExamName,
  loading,
}: SubmissionStatsProps) {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 mb-6">
      <div className="rounded-xl border border-brand-pink-soft bg-brand-pink-soft p-5 shadow-lg">
        <p className="text-xs font-semibold uppercase tracking-[0.4em] text-brand-pink">Total de intentos</p>
        <p className="text-3xl font-extrabold text-brand-pink mt-2">
          {totalSubmissions}
        </p>
        {selectedExamName && (
          <p className="text-xs text-brand-yellow mt-2 opacity-90">Examen: {selectedExamName}</p>
        )}
      </div>
      <div className="rounded-xl border border-brand-blue-soft bg-brand-blue-soft p-5 shadow-lg">
        <p className="text-xs font-semibold uppercase tracking-[0.4em] text-brand-blue">Nota media</p>
        <p className="text-3xl font-extrabold text-brand-blue mt-2">
          {averageScore !== null ? averageScore.toFixed(2) : 'Sin datos'}
        </p>
        {!loading && averageScore === null && (
          <p className="text-xs text-brand-yellow mt-2 opacity-90">Aún no hay notas registradas.</p>
        )}
      </div>
    </div>
  );
}
