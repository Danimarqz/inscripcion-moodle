import { useEffect, useState } from 'preact/hooks';
import { updateMerits } from '../../services/examService';
import { normalizeDni } from '../../utils/validation';

interface MeritsFormProps {
  email: string;
  dni: string;
  examId: number;
  canEditMerits: boolean;
  hasPreviousSubmission: boolean;
  currentMerits: number | null;
  meritsPosition: number | null;
  meritsTotal: number | null;
  aprobados: number | null;
  totalSubmissions: number | null;
}

export default function MeritsForm({
  email,
  dni,
  examId,
  canEditMerits,
  hasPreviousSubmission,
  currentMerits,
  meritsPosition,
  meritsTotal,
  aprobados,
  totalSubmissions,
}: MeritsFormProps) {
  const [meritsValue, setMeritsValue] = useState('');
  const [meritsSubmitting, setMeritsSubmitting] = useState(false);
  const [meritsMessage, setMeritsMessage] = useState<string | null>(null);
  const [localMeritsPosition, setLocalMeritsPosition] = useState<number | null>(null);
  const [localMeritsTotal, setLocalMeritsTotal] = useState<number | null>(null);
  const [meritsSaved, setMeritsSaved] = useState(false);

  useEffect(() => {
    if (currentMerits !== null) setMeritsValue(String(currentMerits));
    setLocalMeritsPosition(meritsPosition);
    setLocalMeritsTotal(meritsTotal);
  }, [currentMerits, meritsPosition, meritsTotal]);

  if (!canEditMerits || !hasPreviousSubmission) return null;

  const effectivePosition = localMeritsPosition ?? meritsPosition;
  const effectiveTotal = localMeritsTotal ?? meritsTotal;
  const meritsLocked = currentMerits !== null || meritsSaved;

  async function handleMeritsSubmit(event: Event) {
    event.preventDefault();
    setMeritsMessage(null);

    const normalizedEmail = email.trim().toLowerCase();
    const normalizedDni = normalizeDni(dni);
    const meritsNum = meritsValue ? parseFloat(meritsValue) : null;

    if (meritsNum !== null && isNaN(meritsNum)) {
      setMeritsMessage('El valor de méritos no es válido.');
      return;
    }

    setMeritsSubmitting(true);
    try {
      const resp = await updateMerits({
        email: normalizedEmail,
        dni: normalizedDni,
        exam_id: examId,
        merits: meritsNum,
      });
      setMeritsMessage(resp.message);
      setMeritsSaved(true);
      if (resp.merits_position != null) setLocalMeritsPosition(resp.merits_position);
      if (resp.merits_total != null) setLocalMeritsTotal(resp.merits_total);
    } catch (error: unknown) {
      setMeritsMessage(error instanceof Error ? error.message : 'Error al guardar méritos');
    } finally {
      setMeritsSubmitting(false);
    }
  }

  return (
    <div className="mt-6 rounded-2xl border border-brand-yellow/60 bg-brand-yellow/5 p-6 shadow-[0_10px_25px_rgba(255,200,50,0.1)]">
      <h3 className="text-lg font-semibold text-brand-yellow mb-3">{meritsLocked ? 'Méritos guardados' : 'Añadir méritos'}</h3>
      <form onSubmit={handleMeritsSubmit} className="flex flex-col gap-3">
        <label className="block text-sm text-gray-300">
          Puntuación de méritos:
          <input
            type="number"
            step="0.01"
            min="0"
            value={meritsValue}
            disabled={meritsLocked}
            onInput={(e) => setMeritsValue((e.target as HTMLInputElement).value)}
            className={`w-full mt-1 px-4 py-3 rounded-lg border border-[#555] text-white transition-all ${meritsLocked ? 'bg-[#2a2d33] opacity-60 cursor-not-allowed' : 'bg-[#1f2229] focus:ring-2 focus:ring-brand-yellow focus:border-transparent'}`}
          />
        </label>
        {!meritsLocked && (
          <button
            type="submit"
            disabled={meritsSubmitting}
            className="btn-brand w-full text-base disabled:opacity-60 disabled:cursor-not-allowed cursor-pointer"
          >
            {meritsSubmitting ? 'Guardando...' : 'Guardar méritos'}
          </button>
        )}
        {meritsMessage && (
          <p className="text-sm text-green-300 bg-green-500/10 border border-green-500/30 p-3 rounded-md">{meritsMessage}</p>
        )}
      </form>
      {effectivePosition != null && effectiveTotal != null && (
        <div className="mt-4 p-3 rounded-lg bg-indigo-500/10 border border-indigo-500/30">
          <p className="text-sm text-indigo-200">CONCURSO - OPOSICION</p>
          <p className="text-sm text-indigo-200">
            Eres el número <span className="font-bold text-indigo-100">{effectivePosition}</span> de{' '}
            <span className="font-bold text-indigo-100">{effectiveTotal}</span> personas aprobadas que han metido sus méritos
          </p>
          <p className="text-sm text-indigo-200">Número de aprobados <span className="font-bold text-indigo-100">{aprobados}</span> de{' '}
            <span className="font-bold text-indigo-100">{totalSubmissions}</span>
            </p>
        </div>
      )}
    </div>
  );
}
