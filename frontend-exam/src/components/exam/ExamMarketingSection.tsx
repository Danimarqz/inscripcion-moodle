interface ExamMarketingSectionProps {
  acceptsMarketing: boolean;
  raffleEnabled: boolean;
  raffleAccepted: boolean;
  raffleTerms: string;
  onMarketingChange: (value: boolean) => void;
  onRaffleChange: (value: boolean) => void;
  onShowRaffleTerms: () => void;
}

export default function ExamMarketingSection({
  acceptsMarketing,
  raffleEnabled,
  raffleAccepted,
  onMarketingChange,
  onRaffleChange,
  onShowRaffleTerms,
}: ExamMarketingSectionProps) {
  return (
    <>
      <div className="mt-8 space-top-3 text-sm">
        <label htmlFor="accepts_marketing" className="flex items-start gap-3 text-gray-200">
          <input
            type="checkbox"
            id="accepts_marketing"
            name="accepts_marketing"
            required
            checked={acceptsMarketing}
            onChange={(event) => onMarketingChange(event.currentTarget.checked)}
            className="mt-1 h-4 w-4 rounded border border-[#555] bg-[#2a2d33] text-brand-pink focus:ring-2 focus:ring-brand-yellow/60"
          />
          <span className="text-gray-300">
            Acepto recibir por email recordatorios y novedades sobre nuevas oposiciones.
          </span>
        </label>
        <p className="text-xs leading-relaxed text-gray-400">
          Al entregar confirmas que utilizaremos tus datos solo para corregir tu simulacro y gestionar el servicio;
          no usamos cookies de seguimiento. Consulta la{' '}
          <a
            href="/politica-de-privacidad"
            className="text-brand-pink underline decoration-dotted hover:text-brand-yellow"
            target="_blank"
            rel="noopener noreferrer"
          >
            Politica de privacidad
          </a>
          .
        </p>
      </div>

      {raffleEnabled && (
        <div className="mt-6 text-sm">
          <label htmlFor="raffle_accepted" className="flex items-start gap-3 text-gray-200">
            <input
              type="checkbox"
              id="raffle_accepted"
              name="raffle_accepted"
              checked={raffleAccepted}
              onChange={(event) => onRaffleChange(event.currentTarget.checked)}
              className="mt-1 h-4 w-4 rounded border border-[#555] bg-[#2a2d33] text-brand-pink focus:ring-2 focus:ring-brand-yellow/60"
            />
            <span className="text-gray-300">
              He leido y acepto las{' '}
              <button
                type="button"
                onClick={onShowRaffleTerms}
                className="text-brand-pink underline decoration-dotted hover:text-brand-yellow"
              >
                bases del sorteo
              </button>
              .
            </span>
          </label>
        </div>
      )}
    </>
  );
}
