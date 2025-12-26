interface PaginationControlsProps {
  currentPage: number;
  totalPages: number;
  onPrev: () => void;
  onNext: () => void;
}

export default function PaginationControls({
  currentPage,
  totalPages,
  onPrev,
  onNext,
}: PaginationControlsProps) {
  return (
    <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between mt-6">
      <button
        type="button"
        disabled={currentPage <= 1}
        onClick={onPrev}
        className={`px-4 py-2 rounded border border-brand-blue-soft text-brand-blue transition-colors ${
          currentPage <= 1
            ? 'opacity-40 cursor-not-allowed'
            : 'hover:bg-[#1b2635] hover:border-brand-pink-soft cursor-pointer'
        }`}
      >
        Anterior
      </button>
      <p className="text-sm text-gray-300">
        Pagina {currentPage} de {totalPages}
      </p>
      <button
        type="button"
        disabled={currentPage >= totalPages}
        onClick={onNext}
        className={`px-4 py-2 rounded border border-brand-blue-soft text-brand-blue transition-colors ${
          currentPage >= totalPages
            ? 'opacity-40 cursor-not-allowed'
            : 'hover:bg-[#1b2635] hover:border-brand-pink-soft cursor-pointer'
        }`}
      >
        Siguiente
      </button>
    </div>
  );
}
