interface SubmissionFilterActionsProps {
  filterMoodleUsers: boolean;
  onFilterChange: (value: boolean) => void;
  downloadingEmails: boolean;
  downloadMessage: string | null;
  downloadError: string | null;
  onDownloadEmails: () => void;
  onComposeEmails: () => void;
  composingEmails: boolean;
  composeMessage: string | null;
  composeError: string | null;
}

export default function SubmissionFilterActions({
  filterMoodleUsers,
  onFilterChange,
  downloadingEmails,
  downloadMessage,
  downloadError,
  onDownloadEmails,
  onComposeEmails,
  composingEmails,
  composeMessage,
  composeError,
}: SubmissionFilterActionsProps) {
  return (
    <div className="flex flex-wrap items-center gap-3 mb-4">
      <label className="flex items-center gap-2 text-xs font-semibold text-white">
        <input
          type="checkbox"
          checked={filterMoodleUsers}
          onInput={(event) => onFilterChange((event.currentTarget as HTMLInputElement).checked)}
          className="h-4 w-4 rounded border border-[#444] bg-[#1f2229] text-brand-blue focus:outline-none focus:ring-2 focus:ring-brand-blue"
        />
        Mostrar solo usuarios con cuenta Moodle
      </label>
      <button
        type="button"
        onClick={onDownloadEmails}
        disabled={downloadingEmails}
        className={`px-4 py-2 rounded text-white transition-colors ${
          downloadingEmails
            ? 'bg-[#1c2230] opacity-60 cursor-progress border border-[#444]'
            : 'bg-brand-pink hover:bg-[#ff8b6d] border border-brand-pink-soft cursor-pointer'
        }`}
        >
        {downloadingEmails ? 'Generando lista...' : 'Descargar emails (.txt)'}
      </button>
      <button
        type="button"
        onClick={onComposeEmails}
        disabled={composingEmails}
        className={`px-4 py-2 rounded text-white transition-colors ${
          composingEmails
            ? 'bg-[#1c2230] opacity-60 cursor-progress border border-[#444]'
            : 'bg-brand-blue hover:bg-[#2ca5ff] border border-brand-blue-soft cursor-pointer'
        }`}
      >
        {composingEmails ? 'Cargando... ' : 'Enviar emails'}
      </button>
      {downloadMessage && <p className="text-xs text-brand-blue w-full md:w-auto">{downloadMessage}</p>}
      {downloadError && <p className="text-xs text-brand-pink w-full md:w-auto">{downloadError}</p>}
      {composeMessage && <p className="text-xs text-brand-blue w-full md:w-auto">{composeMessage}</p>}
      {composeError && <p className="text-xs text-brand-pink w-full md:w-auto">{composeError}</p>}
    </div>
  );
}
