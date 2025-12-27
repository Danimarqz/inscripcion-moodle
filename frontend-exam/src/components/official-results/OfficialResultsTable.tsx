import { useState } from 'preact/hooks';
import type { EditOfficialResultPayload, ExamOfficialResult } from '../../types/exam';
import PaginationControls from './PaginationControls';
import SortableHeader from './SortableHeader';

interface OfficialResultsTableProps {
  selectedExamName: string;
  results: ExamOfficialResult[];
  totalResults: number;
  pageSize: number;
  currentPage: number;
  sortBy: 'dni' | 'nombre' | 'apellidos' | 'usuario' | 'creado';
  sortDirection: 'asc' | 'desc';
  onPageSizeChange: (size: number) => void;
  onPageChange: (page: number) => void;
  onSort: (key: 'dni' | 'nombre' | 'apellidos' | 'usuario' | 'creado') => void;
  onUpdate: (id: number, payload: EditOfficialResultPayload) => Promise<void>;
  onDelete: (id: number) => void | Promise<void>;
}

export default function OfficialResultsTable({
  selectedExamName,
  results,
  totalResults,
  pageSize,
  currentPage,
  sortBy,
  sortDirection,
  onPageSizeChange,
  onPageChange,
  onSort,
  onUpdate,
  onDelete,
}: OfficialResultsTableProps) {
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editPayload, setEditPayload] = useState<EditOfficialResultPayload>({});

  const startEditing = (result: ExamOfficialResult) => {
    setEditingId(result.id);
    setEditPayload({
      dni: result.dni_masked,
      nombre: result.nombre,
      apellido_1: result.apellido_1,
      apellido_2: result.apellido_2 || '',
    });
  };

  const cancelEditing = () => {
    setEditingId(null);
    setEditPayload({});
  };

  const handleSave = async (id: number) => {
    await onUpdate(id, editPayload);
    setEditingId(null);
  };

  return (
    <div className="bg-[#14161d] border border-brand-blue-soft rounded-2xl p-5 shadow-xl">
      <h3 className="text-lg font-semibold text-brand-pink mb-3">
        Listado oficial ({selectedExamName})
      </h3>
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between mb-4">
        <div className="text-sm text-brand-blue font-semibold">Total registros: {totalResults}</div>
        <div className="flex items-center gap-3">
          <label className="text-xs text-brand-yellow flex items-center gap-2">
            Filas por página
            <select
              value={pageSize}
              onChange={(event) => {
                onPageSizeChange(Number((event.target as HTMLSelectElement).value));
              }}
              className="px-2 py-1 rounded border border-brand-yellow-soft bg-[#1f2229] text-white text-xs focus:outline-none focus:ring-2 focus:ring-brand-yellow"
            >
              {[50, 100, 200, 500].map((size) => (
                <option key={size} value={size}>
                  {size}
                </option>
              ))}
            </select>
          </label>
        </div>
      </div>
      <div className="overflow-x-auto">
        <table className="min-w-full text-sm text-left border border-brand-blue-soft rounded-xl overflow-hidden">
          <thead className="bg-[#1f2229] text-brand-yellow uppercase text-xs tracking-[0.3em]">
            <tr>
              <SortableHeader
                label="DNI"
                sortKey="dni"
                activeKey={sortBy}
                direction={sortDirection}
                onSort={onSort}
              />
              <th className="px-3 py-2 text-left cursor-default">Tipo</th>
              <SortableHeader
                label="Nombre"
                sortKey="nombre"
                activeKey={sortBy}
                direction={sortDirection}
                onSort={onSort}
              />
              <SortableHeader
                label="Apellidos"
                sortKey="apellidos"
                activeKey={sortBy}
                direction={sortDirection}
                onSort={onSort}
              />
              <SortableHeader
                label="Usuario asociado"
                sortKey="usuario"
                activeKey={sortBy}
                direction={sortDirection}
                onSort={onSort}
              />
              <SortableHeader
                label="Creado"
                sortKey="creado"
                activeKey={sortBy}
                direction={sortDirection}
                onSort={onSort}
              />
              <th className="px-3 py-2 text-center text-xs">Acciones</th>
            </tr>
          </thead>
          <tbody>
            {results.map((result) => {
              const isEditing = editingId === result.id;
              const surname = [result.apellido_1, result.apellido_2].filter(Boolean).join(' ');
              const user = result.user;
              const associatedUser = user
                ? `${user.name} ${user.surname} (${user.dni}${user.email ? ` - ${user.email}` : ''})`
                : 'Sin usuario enlazado';

              if (isEditing) {
                return (
                  <tr key={result.id} className="bg-[#1f2229] border-b border-brand-blue">
                    <td className="px-3 py-2">
                      <input
                        type="text"
                        value={editPayload.dni}
                        onInput={(e) =>
                          setEditPayload({
                            ...editPayload,
                            dni: (e.target as HTMLInputElement).value,
                          })
                        }
                        className="w-full px-2 py-1 rounded bg-[#14161d] border border-brand-blue text-white text-xs font-mono"
                      />
                    </td>
                     <td className="px-3 py-2">
                       <select
                         value={editPayload.result_type || result.result_type}
                         onChange={(e) =>
                           setEditPayload({
                             ...editPayload,
                             result_type: (e.target as HTMLSelectElement).value,
                           })
                         }
                         className="w-full px-2 py-1 rounded bg-[#14161d] border border-brand-blue text-white text-xs"
                       >
                         <option value="General">General</option>
                         <option value="Promoción interna">Promoción interna</option>
                         <option value="Discapacidad">Discapacidad</option>
                         <option value="Otros">Otros</option>
                       </select>
                    </td>
                    <td className="px-3 py-2">
                      <input
                        type="text"
                        value={editPayload.nombre}
                        onInput={(e) =>
                          setEditPayload({
                            ...editPayload,
                            nombre: (e.target as HTMLInputElement).value,
                          })
                        }
                        className="w-full px-2 py-1 rounded bg-[#14161d] border border-brand-blue text-white text-xs"
                      />
                    </td>
                    <td className="px-3 py-2 flex gap-1">
                      <input
                        type="text"
                        value={editPayload.apellido_1 || ''}
                        placeholder="Apellido 1"
                        onInput={(e) =>
                          setEditPayload({
                            ...editPayload,
                            apellido_1: (e.target as HTMLInputElement).value,
                          })
                        }
                        className="w-1/2 px-2 py-1 rounded bg-[#14161d] border border-brand-blue text-white text-xs"
                      />
                      <input
                        type="text"
                        value={editPayload.apellido_2 || ''}
                        placeholder="Apellido 2"
                        onInput={(e) =>
                          setEditPayload({
                            ...editPayload,
                            apellido_2: (e.target as HTMLInputElement).value,
                          })
                        }
                        className="w-1/2 px-2 py-1 rounded bg-[#14161d] border border-brand-blue text-white text-xs"
                      />
                    </td>
                    <td className="px-3 py-2 text-brand-pink text-xs">{associatedUser}</td>
                    <td className="px-3 py-2 text-xs text-brand-yellow">
                      {new Date(result.created_at).toLocaleString()}
                    </td>
                    <td className="px-3 py-2 text-center whitespace-nowrap">
                      <div className="flex items-center justify-center gap-2">
                        <button
                          onClick={() => handleSave(result.id)}
                          className="px-2 py-1 rounded bg-green-500 text-white text-[10px] font-bold hover:bg-green-600 transition-colors"
                        >
                          GUARDAR
                        </button>
                        <button
                          onClick={cancelEditing}
                          className="px-2 py-1 rounded bg-gray-500 text-white text-[10px] font-bold hover:bg-gray-600 transition-colors"
                        >
                          CANCELAR
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              }

              return (
                <tr
                  key={result.id}
                  className="odd:bg-[#1b1e25] border-b border-brand-blue-soft last:border-b-0 group hover:bg-[#1f2229] transition-colors"
                >
                  <td className="px-3 py-2 font-mono text-xs text-brand-blue">
                    {result.dni_masked}
                  </td>
                  <td className="px-3 py-2 text-white text-xs">{result.result_type || 'General'}</td>
                  <td className="px-3 py-2 text-white font-semibold">{result.nombre}</td>
                  <td className="px-3 py-2 text-white">{surname}</td>
                  <td className="px-3 py-2 text-brand-pink">{associatedUser}</td>
                  <td className="px-3 py-2 text-xs text-brand-yellow">
                    {new Date(result.created_at).toLocaleString()}
                  </td>
                  <td className="px-3 py-2 text-center whitespace-nowrap">
                    <div className="flex items-center justify-center gap-2">
                      <button
                        onClick={() => startEditing(result)}
                        className="p-1 hover:text-brand-blue text-gray-400 transition-colors"
                        title="Editar"
                      >
                        <svg
                          xmlns="http://www.w3.org/2000/svg"
                          className="h-4 w-4"
                          fill="none"
                          viewBox="0 0 24 24"
                          stroke="currentColor"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                          />
                        </svg>
                      </button>
                      <button
                        onClick={() => onDelete(result.id)}
                        className="p-1 hover:text-red-500 text-gray-400 transition-colors"
                        title="Eliminar"
                      >
                        <svg
                          xmlns="http://www.w3.org/2000/svg"
                          className="h-4 w-4"
                          fill="none"
                          viewBox="0 0 24 24"
                          stroke="currentColor"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                          />
                        </svg>
                      </button>
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      <PaginationControls
        totalItems={totalResults}
        pageSize={pageSize}
        currentPage={currentPage}
        onPageChange={onPageChange}
      />
    </div>
  );
}
