type EligibilityModalProps = {
  open: boolean;
  onClose: () => void;
};

export default function EligibilityModal({ open, onClose }: EligibilityModalProps) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 px-4">
      <div className="max-w-md w-full bg-[#161922] border border-brand-blue-soft rounded-2xl p-6 shadow-xl space-y-4">
        <h2 className="text-xl font-bold text-brand-pink">No encontramos tu resultado oficial</h2>
        <p className="text-sm text-gray-200 leading-relaxed">
          Este apartado es solo para personas que realizaron el examen oficial. Si no puedes registrar tus resultados y
          si te presentaste, contacta con nosotros: <span className="text-brand-yellow"><a href="mailto:info@opositatcae.es">info@opositatcae.es</a></span>
        </p>
        <div className="flex justify-end">
          <button type="button" onClick={onClose} className="btn-brand px-5">
            Entendido
          </button>
        </div>
      </div>
    </div>
  );
}
