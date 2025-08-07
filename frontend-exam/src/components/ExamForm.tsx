import { useState } from 'preact/hooks';
import type {
  ExamCreateWithQuestions,
  ExamEdit,
  QuestionCreate,
  QuestionEdit,
} from '../types/exam';
import { getAuthToken } from '../components/AdminDashboard'; // IMPORTA desde donde está definido
import { createExam, editExam } from '../services/adminService';

interface ExamFormProps {
  examToEdit?: ExamEdit & { id: number };
  onSuccess: (examId: number) => void;
}

export default function ExamForm({ examToEdit, onSuccess }: ExamFormProps) {
  const [name, setName] = useState(examToEdit?.name ?? '');
  const [isActive, setIsActive] = useState(examToEdit?.is_active ?? false);
  const [showResponse, setShowResponse] = useState(examToEdit?.show_response ?? false);
  const [questions, setQuestions] = useState<(QuestionCreate | QuestionEdit)[]>(() =>
    examToEdit?.questions.length ? examToEdit.questions : [{ text: '', correct_option: 'A' }]
  );
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  function updateQuestion(index: number, field: keyof QuestionCreate | keyof QuestionEdit, value: string) {
    setQuestions((qs) => {
      const copy = [...qs];
      copy[index] = { ...copy[index], [field]: value };
      return copy;
    });
  }

  function addQuestion() {
    setQuestions((qs) => [...qs, { text: '', correct_option: 'A' }]);
  }

  function removeQuestion(index: number) {
    setQuestions((qs) => qs.filter((_, i) => i !== index));
  }
  async function handleSubmit(e: Event) {
    e.preventDefault();
    setError(null);

    if (!name.trim()) {
      setError('El nombre del examen es obligatorio.');
      return;
    }
    if (questions.length === 0) {
      setError('Debe haber al menos una pregunta.');
      return;
    }
    if (questions.some((q) => !['A', 'B', 'C', 'D'].includes(q.correct_option.toUpperCase()))) {
      setError('Todas las preguntas deben tener una opción correcta válida (A, B, C o D).');
      return;
    }

    const token = getAuthToken();
    if (!token) {
      setError('No autorizado: token no disponible.');
      return;
    }

    setLoading(true);

    try {
      const body: ExamCreateWithQuestions | ExamEdit = {
        name: name.trim(),
        is_active: isActive,
        show_response: showResponse,
        questions: questions.map((q) => ({
          id: 'id' in q ? q.id : undefined,
          correct_option: q.correct_option.toUpperCase(),
        })),
      };
    let data;
    if (examToEdit) {
      data = await editExam(examToEdit.id, body, token);
    } else {
      data = await createExam(body as ExamCreateWithQuestions, token);
    }

    onSuccess(data.id);
  } catch (err) {
    setError((err as Error).message || 'Error desconocido');
  } finally {
    setLoading(false);
  }
  }

  return (
    <form onSubmit={handleSubmit} className="exam-form">
      <h2>{examToEdit ? 'Editar Examen' : 'Crear Examen'}</h2>

      {error && <p className="error-message">{error}</p>}

      <label className="form-label">
        Nombre del examen:
        <input
          type="text"
          value={name}
          onInput={(e) => setName((e.target as HTMLInputElement).value)}
          required
          className="form-input"
        />
      </label>

      <label className="form-checkbox-label">
        <input
          type="checkbox"
          checked={isActive}
          onChange={(e) => setIsActive(e.currentTarget.checked)}
          className="form-checkbox"
        />
        Activo
      </label>

      <label className="form-checkbox-label">
        <input
          type="checkbox"
          checked={showResponse}
          onChange={(e) => setShowResponse(e.currentTarget.checked)}
          className="form-checkbox"
        />
        Mostrar respuestas
      </label>

      <fieldset className="question-fieldset">
        <legend>Preguntas</legend>
        {questions.map((q, i) => (
          <div key={i} className="question-block">
            <label className="form-label">
              Opción correcta:
              <select
                value={q.correct_option.toUpperCase()}
                onChange={(e) => updateQuestion(i, 'correct_option', e.currentTarget.value)}
                className="form-select"
              >
                <option value="A">A</option>
                <option value="B">B</option>
                <option value="C">C</option>
                <option value="D">D</option>
              </select>
            </label>
            <button
              type="button"
              onClick={() => removeQuestion(i)}
              disabled={questions.length === 1}
              className="delete-question-button"
              aria-label={`Eliminar pregunta ${i + 1}`}
            >
              Eliminar pregunta
            </button>
          </div>
        ))}
        <button type="button" onClick={addQuestion} className="add-question-button">
          Añadir pregunta
        </button>
      </fieldset>

      <button type="submit" disabled={loading} className="submit-button">
        {loading ? (examToEdit ? 'Guardando...' : 'Creando...') : (examToEdit ? 'Guardar cambios' : 'Crear examen')}
      </button>
    </form>
  );
}
