import { useEffect, useState } from 'preact/hooks';
import Card from './Card';
import type { Exam } from '../types/exam';
import { getExams } from '../services/examService';

export default function IndexPage() {
  const [exams, setExams] = useState<Exam[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function fetchExams() {
      try {
        const data = await getExams();

        if (!Array.isArray(data)) {
          throw new Error('La respuesta de la API no es un array de exámenes.');
        }

        setExams(data);
      } catch (e: any) {
        console.error('Error fetching exams:', e);
        setError('No se pudieron cargar los exámenes. Por favor, inténtalo de nuevo más tarde.');
      } finally {
        setLoading(false);
      }
    }
    fetchExams();
  }, []);

  if (loading) return <p>Cargando exámenes...</p>;

  return (
    <>
      <h1>Bienvenid@ a <span class="text-gradient">OpositaTest</span></h1>
      <p>Elige tu examen:</p>
      {error && <p class="error">{error}</p>}

      <section class="exams-section">
        {exams.length === 0 && !error ? (
          <p class="no-exams">No hay exámenes disponibles en este momento.</p>
        ) : (
          <ul role="list" class="link-card-grid">
            {exams.map((exam) => (
              <Card
                href={`/exam/${exam.id}`}
                title={exam.name}
                body="Comienza tu examen de oposición."
              />
            ))}
          </ul>
        )}
      </section>
    </>
  );
}
