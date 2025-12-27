import type { SubmissionOrderBy, SubmissionOrderDir } from './types';

interface SubmissionFiltersProps {
  searchTerm: string;
  onSearchTermChange: (value: string) => void;
  orderBy: SubmissionOrderBy;
  onOrderByChange: (value: SubmissionOrderBy) => void;
  orderDir: SubmissionOrderDir;
  onOrderDirChange: (value: SubmissionOrderDir) => void;
  filterType: string;
  onFilterTypeChange: (value: string) => void;
}

export default function SubmissionFilters({
  searchTerm,
  onSearchTermChange,
  orderBy,
  onOrderByChange,
  orderDir,
  onOrderDirChange,
  filterType,
  onFilterTypeChange,
}: SubmissionFiltersProps) {
  return (
    <div className="grid gap-4 md:grid-cols-4 mb-4">
      <label className="flex flex-col gap-2">
        <span className="text-xs font-semibold uppercase tracking-[0.35em] text-brand-blue">Buscar</span>
        <input
          type="text"
          placeholder="Nombre, email o DNI"
          value={searchTerm}
          onInput={(event) => onSearchTermChange((event.currentTarget as HTMLInputElement).value)}
          className="w-full px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-brand-blue focus:border-brand-blue"
        />
      </label>
      <label className="flex flex-col gap-2">
         <span className="text-xs font-semibold uppercase tracking-[0.35em] text-brand-blue">Tipo</span>
         <select
           value={filterType}
           onChange={(event) => onFilterTypeChange(event.currentTarget.value)}
           className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue focus:border-brand-blue"
         >
           <option value="">Todos</option>
           <option value="General">General</option>
           <option value="Promoción interna">Promoción interna</option>
           <option value="Discapacidad">Discapacidad</option>
           <option value="Otros">Otros</option>
         </select>
       </label>
      <label className="flex flex-col gap-2">
        <span className="text-xs font-semibold uppercase tracking-[0.35em] text-brand-blue">Ordenar por</span>
        <select
          value={orderBy}
          onChange={(event) => onOrderByChange(event.currentTarget.value as SubmissionOrderBy)}
          className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue focus:border-brand-blue"
        >
          <option value="submitted_at">Fecha envío</option>
          <option value="score">Nota</option>
          <option value="name">Nombre</option>
          <option value="surname">Apellido</option>
        </select>
      </label>
      <label className="flex flex-col gap-2">
        <span className="text-xs font-semibold uppercase tracking-[0.35em] text-brand-blue">Dirección</span>
        <select
          value={orderDir}
          onChange={(event) => onOrderDirChange(event.currentTarget.value as SubmissionOrderDir)}
          className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue focus:border-brand-blue"
        >
          <option value="desc">Descendente</option>
          <option value="asc">Ascendente</option>
        </select>
      </label>
    </div>
  );
}
