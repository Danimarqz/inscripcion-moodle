import { useEffect, useState } from 'preact/hooks';

import { deleteExam, getAdminExams, validateAdminToken } from '../services/adminService';
import type { Exam } from '../types/exam';
import SubmissionsManager from './SubmissionsManager';

export function getAuthToken(): string | null {
  return localStorage.getItem('admin_access_token');
}

export function removeAuthToken() {
  localStorage.removeItem('admin_access_token');
}

export default function AdminDashboard() {
  const [exams, setExams] = useState<Exam[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const rawToken = getAuthToken();
  if (!rawToken) {
    window.location.href = '/admin/login';
    return null;
  }
  const authToken: string = rawToken;

  useEffect(() => {
    async function initialize() {
      const isValid = await validateAdminToken(authToken);
      if (!isValid) {
        removeAuthToken();
        window.location.href = '/admin/login';
        return;
      }

      try {
        const data = await getAdminExams(authToken);
        setExams(data);
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : String(e));
      } finally {
        setLoading(false);
      }
    }

    initialize();
  }, [authToken]);

  async function handleDelete(id: number) {
    if (!confirm('Seguro que quieres borrar este examen?')) return;
    try {
      await deleteExam(id, authToken);
      setExams((current) => current.filter((exam) => exam.id !== id));
      alert('Examen borrado');
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : 'Error desconocido';
      alert('Error borrando examen: ' + message);
    }
  }

  function handleLogout() {
    removeAuthToken();
    window.location.href = '/admin/login';
  }

  if (loading) return <p>Cargando examenes...</p>;

  return (
    <div className="max-w-4xl mx-auto my-8 p-8 bg-[#1a1c22] rounded-lg shadow-2xl text-white">
      <div className="text-center mt-8">
        <button
          className="bg-gray-700 border-none py-2 px-5 rounded text-white cursor-pointer transition-colors duration-300 hover:bg-gray-800"
          onClick={handleLogout}
        >
          Cerrar sesion
        </button>
      </div>

      {error && (
        <p className="text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md my-6">
          Error: {error}
        </p>
      )}

      <h2 className="text-purple-300 border-b-2 border-purple-500 pb-2 mb-6">Examenes disponibles</h2>

      <div className="mb-6 text-right">
        <button
          className="py-2 px-5 rounded font-semibold cursor-pointer transition-all duration-300 no-underline inline-block border-none text-white bg-purple-600 hover:bg-purple-700 hover:shadow-lg hover:-translate-y-0.5"
          onClick={() => (window.location.href = '/admin/exam/create')}
        >
          Crear nuevo examen
        </button>
      </div>

      {exams.length === 0 ? (
        <p>No hay examenes disponibles.</p>
      ) : (
        <ul className="list-none p-0 m-0">
          {exams.map((exam: Exam) => (
            <li
              key={exam.id}
              className="bg-[#2a2d34] p-4 my-4 rounded flex justify-between items-center transition-colors duration-300 hover:bg-[#3a3d44]"
            >
              <span>{exam.name} (ID: {exam.id})</span>
              <div className="flex gap-3">
                <button
                  className="py-2 px-5 rounded font-semibold cursor-pointer transition-all duration-300 no-underline inline-block border-none text-white bg-purple-600 hover:bg-purple-700 hover:shadow-lg hover:-translate-y-0.5"
                  onClick={() => (window.location.href = `/admin/exam/${exam.id}/edit`)}
                >
                  Editar
                </button>
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

      <SubmissionsManager exams={exams} token={authToken} />
    </div>
  );
}
