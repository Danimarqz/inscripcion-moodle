import { normalizeDni } from '../../utils/validation';

interface QuickResultCheckProps {
  email: string;
  dni: string;
  checking: boolean;
  onEmailChange: (value: string) => void;
  onDniChange: (value: string) => void;
  onSubmit: (event: Event) => void;
}

export default function QuickResultCheck({
  email,
  dni,
  checking,
  onEmailChange,
  onDniChange,
  onSubmit,
}: QuickResultCheckProps) {
  return (
    <section className="mb-10 rounded-2xl border border-brand-blue-soft bg-brand-blue-soft p-6 shadow-[0_10px_30px_rgba(15,153,188,0.2)]">
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 mb-4">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.4em] text-brand-yellow">Consulta rapida</p>
          <h2 className="text-2xl font-bold text-brand-pink">Consultar mi nota</h2>
          <p className="text-sm text-brand-blue text-opacity-80 mt-1">
            Introduce el mismo email y DNI/NIE con el que enviaste el intento para ver tu resultado.
          </p>
        </div>
      </div>
      <form className="flex flex-col gap-4 md:flex-row md:items-end" onSubmit={onSubmit} noValidate>
        <label className="flex-1 text-sm font-semibold text-brand-pink">
          Email
          <input
            type="email"
            value={email}
            onInput={(event) => onEmailChange((event.target as HTMLInputElement).value)}
            className="mt-1 w-full rounded border border-[#5a4a7a] bg-[#1d1f27] px-3 py-2 text-white focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/40"
            placeholder="tu@email.com"
            required
          />
        </label>
        <label className="flex-1 text-sm font-semibold text-brand-pink">
          DNI / NIE
          <input
            type="text"
            value={dni}
            onInput={(event) => onDniChange(normalizeDni((event.target as HTMLInputElement).value))}
            className="mt-1 w-full rounded border border-[#5a4a7a] bg-[#1d1f27] px-3 py-2 text-white uppercase focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/40"
            placeholder="00000000A"
            required
          />
        </label>
        <button type="submit" className="btn-brand w-full md:w-auto px-6" disabled={checking}>
          {checking ? 'Consultando...' : 'Consultar mi nota'}
        </button>
      </form>
    </section>
  );
}
