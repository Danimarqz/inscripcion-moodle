import { useEffect, useState } from 'preact/hooks';
import { checkOfficialResult } from '../services/examService';
import { normalizeDni, validateDniNie } from '../utils/validation';

export function useEligibilityCheck(name: string, surname: string, dni: string, examId: number) {
  const [allowed, setAllowed] = useState(false);
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showModal, setShowModal] = useState(false);

  // Reset on examId change
  useEffect(() => {
    setAllowed(false);
    setError(null);
    setShowModal(false);
  }, [examId]);

  // Debounced eligibility check
  useEffect(() => {
    setAllowed(false);
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

  return { allowed, checking, error, showModal, setShowModal, setAllowed };
}
