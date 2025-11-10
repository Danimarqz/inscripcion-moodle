import { useEffect, useState } from 'preact/hooks';

import { deleteExam, getAdminExams } from '../services/adminService';
import type { Exam } from '../types/exam';
import { useAdminAuth } from '../hooks/useAdminAuth';
import { removeAuthToken, redirectToLogin } from '../utils/adminAuth';

export default function AdminExamsManager() {
  const { token, loading: authenticating, error: authError } = useAdminAuth();
  const [exams, setExams] = useState<Exam[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function fetchExams(currentToken: string) {
      setLoading(true);
      setError(null);
      try {
        const data = await getAdminExams(currentToken);
        if (!cancelled) {
          setExams(data);
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
      fetchExams(token);
    }

    return () => {
      cancelled = true;
    };
  }, [authenticating, token]);

  async function handleDelete(id: number) {
    if (!token) {
      redirectToLogin();
      return;
    }

    if (!confirm('Seguro que quieres borrar este examen?')) return;

    try {
      await deleteExam(id, token);
      setExams((current) => current.filter((exam) => exam.id !== id));
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  if (authenticating) {
    return <p>Cargando examenes...</p>;
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

      {loading ? (
        <p>Cargando examenes...</p>
      ) : exams.length === 0 ? (
        <p>No hay examenes disponibles.</p>
      ) : (
        <ul className="list-none p-0 m-0">
          {exams.map((exam) => (
            <li
              key={exam.id}
              className="bg-[#1f232b] border border-brand-blue-soft p-4 my-4 rounded-xl flex flex-col gap-4 md:flex-row md:items-center md:justify-between shadow-lg transition-all duration-300 hover:bg-brand-yellow-soft hover:border-brand-yellow hover:shadow-[0_18px_40px_rgba(250,164,11,0.25)]"
            >
              <div>
                <p className="font-semibold text-lg text-brand-blue">{exam.name}</p>
                <p className="text-xs font-semibold text-brand-yellow opacity-90 mt-1">
                  ID oficial #{exam.id}
                </p>
              </div>
              <div className="flex gap-3 flex-wrap">
                <a
                  className="py-2 px-5 rounded font-semibold cursor-pointer transition-all duration-300 no-underline inline-block border-none text-dark-200 bg-brand-pink hover:bg-brand-yellow hover:-translate-y-0.5"
                  href={`/admin/exam/${exam.id}/edit`}
                >
                  Editar
                </a>
                <button
                  className="py-2 px-5 rounded font-semibold cursor-pointer transition-all duration-300 no-underline inline-block border-none text-white bg-red-600 hover:bg-red-700 hover:shadow-lg hover:-translate-y-0.5"
                  onClick={() => handleDelete(exam.id)}
                >
                  Borrar
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
      </>
  );
}
