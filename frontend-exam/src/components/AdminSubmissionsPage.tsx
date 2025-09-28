import { useEffect, useState } from 'preact/hooks';

import { getAdminExams } from '../services/adminService';
import type { Exam } from '../types/exam';
import { useAdminAuth } from '../hooks/useAdminAuth';
import SubmissionsManager from './SubmissionsManager';

export default function AdminSubmissionsPage() {
  const { token, loading: authenticating, error: authError } = useAdminAuth();
  const [exams, setExams] = useState<Exam[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function fetchData(currentToken: string) {
      setLoading(true);
      setError(null);
      try {
        const examList = await getAdminExams(currentToken);
        if (!cancelled) {
          setExams(examList);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : String(err));
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    if (!authenticating && token) {
      fetchData(token);
    }

    return () => {
      cancelled = true;
    };
  }, [authenticating, token]);

  if (authenticating) {
    return <p>Cargando gestion de intentos...</p>;
  }

  if (!token) {
    return null;
  }

  return (
    <>
      {authError && (
        <p className="text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md my-6">
          Error de autenticacion: {authError}
        </p>
      )}

      {error && (
        <p className="text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md my-6">Error: {error}</p>
      )}

      {loading ? <p>Cargando examenes...</p> 
      : !error ? <SubmissionsManager exams={exams} token={token} /> 
      : <p>No hay exámenes disponibles</p>}
    </>
  );
}
