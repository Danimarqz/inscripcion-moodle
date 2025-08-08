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
      <h1 class="text-5xl font-extrabold leading-tight text-center mb-12 text-purple-300 shadow-purple-500/50">
        Bienvenid@ a <span class="text-gradient">OpositaTest</span>
      </h1>
      <p class="text-center text-xl mb-8">Elige tu examen:</p>
      {error && <p class="text-center text-lg mt-8 p-4 rounded-lg border border-red-500 text-red-500 bg-red-500/10">{error}</p>}

      <section>
        {exams.length === 0 && !error ? (
          <p class="text-center text-lg mt-8 p-4 rounded-lg border border-green-500 text-green-500 bg-green-500/10">No hay exámenes disponibles en este momento.</p>
        ) : (
          <ul role="list" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8 p-0 list-none">
            {exams.map((exam) => (
              <Card
                key={exam.id}
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
