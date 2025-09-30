import { useAdminAuth } from '../hooks/useAdminAuth';
import { useAdminExams } from '../hooks/useAdminExams';
import OfficialResultsManager from './OfficialResultsManager';

export default function AdminResultsPage() {
  const { token, loading: authenticating, error: authError } = useAdminAuth();
  const { exams, loading, error } = useAdminExams(token, !authenticating && !!token);

  if (authenticating) {
    return <p>Cargando resultados oficiales...</p>;
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

      {loading ? (
        <p>Cargando examenes...</p>
      ) : !error ? (
        <OfficialResultsManager exams={exams} token={token} />
      ) : (
        <p>No hay examenes disponibles</p>
      )}
    </>
  );
}
