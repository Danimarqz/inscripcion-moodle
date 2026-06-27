type RaffleTermsModalProps = {
  open: boolean;
  terms: string;
  onClose: () => void;
};

export default function RaffleTermsModal({ open, terms, onClose }: RaffleTermsModalProps) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 px-4">
      <div className="max-w-lg w-full max-h-[80vh] overflow-y-auto bg-[#161922] border border-brand-blue-soft rounded-2xl p-6 shadow-xl space-y-4">
        <h2 className="text-xl font-bold text-brand-pink">Bases del sorteo</h2>
        <p className="text-sm text-gray-200 leading-relaxed whitespace-pre-wrap">{terms}</p>
        <div className="flex justify-end">
          <button type="button" onClick={onClose} className="btn-brand px-5">
            Cerrar
          </button>
        </div>
      </div>
    </div>
  );
}
