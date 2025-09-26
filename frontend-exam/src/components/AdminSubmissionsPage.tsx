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
    <div className="max-w-5xl mx-auto my-8 p-8 bg-[#1a1c22] rounded-lg shadow-2xl text-white">
      <header className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-purple-300">Gestion de intentos</h1>
          <p className="text-sm text-gray-400">Revisa y actualiza envios de los usuarios.</p>
        </div>
      </header>

      {authError && (
        <p className="text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md my-6">
          Error de autenticacion: {authError}
        </p>
      )}

      {error && (
        <p className="text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md my-6">Error: {error}</p>
      )}

      {loading ? <p>Cargando examenes...</p> : <SubmissionsManager exams={exams} token={token} />}
    </div>
  );
}
