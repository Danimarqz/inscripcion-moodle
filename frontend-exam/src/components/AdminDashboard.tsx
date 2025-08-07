import { useEffect, useState } from 'preact/hooks';
import { getExams } from '../services/examService';
import { deleteExam } from '../services/adminService';
import type { Exam } from '../types/exam';

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
    if (!authToken) {
      window.location.href = '/admin/login';
      return;
    }

    async function fetchExams() {
      try {
        const data = await getExams();
        setExams(data);
      } catch (e: unknown) {
        if (e instanceof Error) {
          setError(e.message);
        } else {
          setError(String(e));
        }
      } finally {
        setLoading(false);
      }
    }

    fetchExams();
  }, []);

  async function handleDelete(id: number) {
    if (!confirm('¿Seguro que quieres borrar?')) return;
    try {
      await deleteExam(id, authToken);
      setExams((exams) => exams.filter((exam) => exam.id !== id));
      alert('Examen borrado');
    } catch (e: unknown) {
      if (e instanceof Error) {
        alert('Error borrando examen: ' + e.message);
      } else {
        alert('Error borrando examen');
      }
    }
  }

  function handleLogout() {
    removeAuthToken();
    window.location.href = '/admin/login';
  }

  if (loading) return <p>Cargando exámenes...</p>;

  return (
    <div className="admin-dashboard">
      <div className="logout-container">
        <button className="submit-button logout-button" onClick={handleLogout}>
          Cerrar Sesión
        </button>
      </div>

      {error && <p className="error-message">Error: {error}</p>}

      <h2>Exámenes Disponibles</h2>

      <button
        className="submit-button"
        onClick={() => window.location.href = '/admin/exam/create'}
      >
        Crear Nuevo Examen
      </button>

      {exams.length === 0 ? (
        <p>No hay exámenes disponibles.</p>
      ) : (
        <ul>
          {exams.map((exam: Exam) => (
            <li key={exam.id}>
              {exam.name} (ID: {exam.id}){' '}
              <button onClick={() => (window.location.href = `/admin/exam/${exam.id}/edit`)}>
                Editar
              </button>{' '}
              <button onClick={() => handleDelete(exam.id)}>Borrar</button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
