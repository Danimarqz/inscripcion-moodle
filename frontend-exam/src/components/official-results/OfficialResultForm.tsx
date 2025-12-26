import { useState } from 'preact/hooks';
import { createOfficialResult } from '../../services/adminService';
import type { CreateOfficialResultPayload } from '../../types/exam';

interface OfficialResultFormProps {
  examId: string;
  token: string;
  onSuccess: (message: string) => void;
  onError: (error: string) => void;
}

export default function OfficialResultForm({ examId, token, onSuccess, onError }: OfficialResultFormProps) {
  const [creating, setCreating] = useState(false);
  const [manualResult, setManualResult] = useState<{
    dni: string;
    apellido1: string;
    apellido2: string;
    nombre: string;
  }>({
    dni: '',
    apellido1: '',
    apellido2: '',
    nombre: '',
  });

  async function handleCreateResult(event: Event) {
    event.preventDefault();
    if (!examId) {
      onError('Selecciona un examen antes de añadir un registro.');
      return;
    }

    const examNumericId = Number(examId);
    if (Number.isNaN(examNumericId)) {
      onError('Identificador de examen inválido.');
      return;
    }

    const dni = manualResult.dni.trim();
    const apellido1 = manualResult.apellido1.trim();
    const apellido2 = manualResult.apellido2.trim();
    const nombre = manualResult.nombre.trim();

    if (!dni || !apellido1 || !nombre) {
      onError('DNI, apellido 1 y nombre son obligatorios.');
      return;
    }

    const payload: CreateOfficialResultPayload = {
      dni,
      apellido_1: apellido1,
      nombre,
      ...(apellido2 ? { apellido_2: apellido2 } : {}),
    };

    setCreating(true);
    onError(''); // Clear error

    try {
      await createOfficialResult(examNumericId, payload, token);
      setManualResult({ dni: '', apellido1: '', apellido2: '', nombre: '' });
      onSuccess('Registro agregado correctamente.');
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err));
    } finally {
      setCreating(false);
    }
  }

  return (
    <form
      className="grid gap-3 md:grid-cols-[repeat(5,minmax(0,1fr))] items-end bg-[#14161d] border border-brand-blue-soft rounded-2xl p-4 shadow-xl"
      onSubmit={handleCreateResult}
    >
      <div className="flex flex-col gap-1">
        <label className="text-xs text-brand-pink font-semibold uppercase tracking-[0.25em]">DNI</label>
        <input
          type="text"
          value={manualResult.dni}
          onInput={(event) =>
            setManualResult((prev) => ({ ...prev, dni: (event.target as HTMLInputElement).value }))
          }
          className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue"
          placeholder="00000000A"
          disabled={!examId || creating}
        />
      </div>
      <div className="flex flex-col gap-1">
        <label className="text-xs text-brand-pink font-semibold uppercase tracking-[0.25em]">
          Apellido 1
        </label>
        <input
          type="text"
          value={manualResult.apellido1}
          onInput={(event) =>
            setManualResult((prev) => ({
              ...prev,
              apellido1: (event.target as HTMLInputElement).value,
            }))
          }
          className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue"
          placeholder="Primer apellido"
          disabled={!examId || creating}
        />
      </div>
      <div className="flex flex-col gap-1">
        <label className="text-xs text-brand-pink font-semibold uppercase tracking-[0.25em]">
          Apellido 2 (opcional)
        </label>
        <input
          type="text"
          value={manualResult.apellido2}
          onInput={(event) =>
            setManualResult((prev) => ({
              ...prev,
              apellido2: (event.target as HTMLInputElement).value,
            }))
          }
          className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue"
          placeholder="Segundo apellido"
          disabled={!examId || creating}
        />
      </div>
      <div className="flex flex-col gap-1">
        <label className="text-xs text-brand-pink font-semibold uppercase tracking-[0.25em]">Nombre</label>
        <input
          type="text"
          value={manualResult.nombre}
          onInput={(event) =>
            setManualResult((prev) => ({ ...prev, nombre: (event.target as HTMLInputElement).value }))
          }
          className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue"
          placeholder="Nombre"
          disabled={!examId || creating}
        />
      </div>
      <div className="flex md:justify-end">
        <button
          type="submit"
          className="w-full md:w-auto py-2 px-5 rounded font-semibold bg-brand-pink text-white shadow-[0_10px_30px_rgba(237,87,150,0.25)] hover:bg-[#f3529f] transition-colors cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed"
          disabled={!examId || creating}
        >
          {creating ? 'Añadiendo...' : 'Añadir registro'}
        </button>
      </div>
    </form>
  );
}
