import { useEffect, useState } from 'preact/hooks';
import { checkOfficialResult } from '../services/examService';
import { normalizeDni, validateDniNie } from '../utils/validation';

export function useEligibilityCheck(name: string, surname: string, dni: string, examId: number) {
  const [allowed, setAllowed] = useState(false);
  const [hasOfficialScore, setHasOfficialScore] = useState(false);
  const [hasOfficialMerits, setHasOfficialMerits] = useState(false);
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showModal, setShowModal] = useState(false);

  // Reset on examId change
  useEffect(() => {
    setAllowed(false);
    setHasOfficialScore(false);
    setHasOfficialMerits(false);
    setError(null);
    setShowModal(false);
  }, [examId]);

  // Debounced eligibility check
  useEffect(() => {
    // Los flags oficiales se resetean junto con `allowed`: si al editar el DNI
    // se pasa de alguien con nota oficial a alguien sin ella, dejarlos puestos
    // mantendría las preguntas ocultas.
    setAllowed(false);
    setHasOfficialScore(false);
    setHasOfficialMerits(false);
    setError(null);
    setShowModal(false);

    const trimmedName = name.trim();
    const trimmedSurname = surname.trim();
    const normalizedDni = normalizeDni(dni);

    if (!trimmedName || !trimmedSurname) {
      setChecking(false);
      return;
    }
    if (!validateDniNie(normalizedDni)) {
      setChecking(false);
      return;
    }

    const abortController = new AbortController();
    setChecking(true);
    const timer = window.setTimeout(() => {
      void checkOfficialResult({
        exam_id: examId,
        name: trimmedName,
        surname: trimmedSurname,
        dni: normalizedDni,
      }, abortController.signal)
        .then((res) => {
          if (abortController.signal.aborted) return;
          setAllowed(res.match);
          setHasOfficialScore(res.match && (res.has_official_score ?? false));
          setHasOfficialMerits(res.match && (res.has_official_merits ?? false));
          if (!res.match) {
            setShowModal(true);
          }
        })
        .catch((err) => {
          if (abortController.signal.aborted) return;
          setError(err instanceof Error ? err.message : 'No se pudo comprobar el acceso');
        })
        .finally(() => {
          if (abortController.signal.aborted) return;
          setChecking(false);
        });
    }, 350);

    return () => {
      abortController.abort();
      window.clearTimeout(timer);
      setChecking(false);
    };
  }, [name, surname, dni, examId]);

  return { allowed, hasOfficialScore, hasOfficialMerits, checking, error, showModal, setShowModal, setAllowed };
}
