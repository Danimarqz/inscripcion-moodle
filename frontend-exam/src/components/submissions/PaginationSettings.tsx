interface PaginationSettingsProps {
  pageLimit: number;
  onLimitChange: (value: number) => void;
  pageInput: number;
  onPageInputChange: (value: number) => void;
  totalPages: number;
  goToPage: () => void;
}

export default function PaginationSettings({
  pageLimit,
  onLimitChange,
  pageInput,
  onPageInputChange,
  totalPages,
  goToPage,
}: PaginationSettingsProps) {
  return (
    <div className="grid gap-4 md:grid-cols-3 mb-6">
      <label className="flex flex-col gap-2">
        <span className="text-xs font-semibold uppercase tracking-[0.35em] text-brand-blue">Mostrar por página</span>
        <select
          value={pageLimit}
          onChange={(event) => onLimitChange(Number(event.currentTarget.value))}
          className="px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue focus:border-brand-blue"
        >
          {[10, 25, 50, 100].map((value) => (
            <option key={value} value={value}>
              {value}
            </option>
          ))}
        </select>
      </label>
      <label className="flex flex-col gap-2">
        <span className="text-xs font-semibold uppercase tracking-[0.35em] text-brand-blue">Ir a página</span>
        <div className="flex gap-2">
          <input
            type="number"
            min={1}
            max={totalPages}
            value={pageInput}
            onInput={(event) => onPageInputChange(Number(event.currentTarget.value))}
            className="w-full px-3 py-2 rounded border border-[#444] bg-[#1f2229] text-white focus:outline-none focus:ring-2 focus:ring-brand-blue focus:border-brand-blue"
          />
          <button
            type="button"
            onClick={goToPage}
            className="px-4 py-2 rounded cursor-pointer border border-brand-blue-soft text-brand-blue transition-colors hover:bg-[#1b2635] hover:border-brand-pink-soft"
          >
            Ir
          </button>
        </div>
      </label>
      <div className="flex flex-col justify-end">
        <p className="text-xs text-gray-400">
          de {totalPages} {totalPages === 1 ? 'página' : 'páginas'}
        </p>
      </div>
    </div>
  );
}
