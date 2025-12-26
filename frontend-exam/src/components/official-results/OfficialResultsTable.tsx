import type { ExamOfficialResult } from '../../types/exam';
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
}: OfficialResultsTableProps) {
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
                label="DNI Excel"
                sortKey="dni"
                activeKey={sortBy}
                direction={sortDirection}
                onSort={onSort}
              />
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
            </tr>
          </thead>
          <tbody>
            {results.map((result) => {
              const surname = [result.apellido_1, result.apellido_2].filter(Boolean).join(' ');
              const user = result.user;
              const associatedUser = user
                ? `${user.name} ${user.surname} (${user.dni}${user.email ? ` - ${user.email}` : ''})`
                : 'Sin usuario enlazado';

              return (
                <tr
                  key={result.id}
                  className="odd:bg-[#1b1e25] border-b border-brand-blue-soft last:border-b-0"
                >
                  <td className="px-3 py-2 font-mono text-xs text-brand-blue">
                    {result.dni_masked}
                  </td>
                  <td className="px-3 py-2 text-white font-semibold">{result.nombre}</td>
                  <td className="px-3 py-2 text-white">{surname}</td>
                  <td className="px-3 py-2 text-brand-pink">{associatedUser}</td>
                  <td className="px-3 py-2 text-xs text-brand-yellow">
                    {new Date(result.created_at).toLocaleString()}
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
