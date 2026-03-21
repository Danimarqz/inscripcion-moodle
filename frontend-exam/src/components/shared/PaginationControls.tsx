interface PaginationControlsProps {
  totalItems: number;
  pageSize: number;
  currentPage: number;
  onPageChange: (page: number) => void;
}

export default function PaginationControls({
  totalItems,
  pageSize,
  currentPage,
  onPageChange,
}: PaginationControlsProps) {
  const totalPages = Math.max(1, Math.ceil(totalItems / pageSize));
  const startItem = Math.min(totalItems, (currentPage - 1) * pageSize + 1);
  const endItem = Math.min(totalItems, currentPage * pageSize);

  if (totalItems <= pageSize) {
    return null;
  }

  return (
    <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between mt-4 text-xs text-brand-blue">
      <div className="font-semibold">
        Mostrando {startItem}-{endItem} de {totalItems} registros - Página {currentPage} de {totalPages}
      </div>
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={() => onPageChange(Math.max(1, currentPage - 1))}
          className="px-3 py-1 rounded border border-brand-blue-soft text-brand-blue hover:bg-brand-blue-soft disabled:opacity-50 disabled:cursor-not-allowed"
          disabled={currentPage <= 1}
        >
          Anterior
        </button>
        <button
          type="button"
          onClick={() => onPageChange(Math.min(totalPages, currentPage + 1))}
          className="px-3 py-1 rounded border border-brand-blue-soft text-brand-blue hover:bg-brand-blue-soft disabled:opacity-50 disabled:cursor-not-allowed"
          disabled={currentPage >= totalPages}
        >
          Siguiente
        </button>
      </div>
    </div>
  );
}
