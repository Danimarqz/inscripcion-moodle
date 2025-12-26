import type { JSX } from 'preact';
import { useEffect, useRef, useState } from 'preact/hooks';

export type EmailCandidate = {
  email: string;
  selected: boolean;
};

interface EmailComposerProps {
  loading: boolean;
  sending: boolean;
  subject: string;
  body: string;
  attachments: File[];
  candidates: EmailCandidate[];
  selectedCount: number;
  error?: string | null;
  onSubjectChange: (value: string) => void;
  onBodyChange: (value: string) => void;
  onAddAttachments: (files: FileList | null) => void;
  onRemoveAttachment: (index: number) => void;
  onToggleRecipient: (index: number) => void;
  onClose: () => void;
  onSend: () => void;
}

export default function EmailComposer({
  loading,
  sending,
  subject,
  body,
  attachments,
  candidates,
  selectedCount,
  error,
  onSubjectChange,
  onBodyChange,
  onAddAttachments,
  onRemoveAttachment,
  onToggleRecipient,
  onClose,
  onSend,
}: EmailComposerProps) {
  const [dragPosition, setDragPosition] = useState<{ x: number; y: number } | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const dialogRef = useRef<HTMLDivElement>(null);
  const dragOffsetRef = useRef<{ x: number; y: number } | null>(null);

  const handlePointerMove = (event: PointerEvent) => {
    const offset = dragOffsetRef.current;
    if (!offset) {
      return;
    }
    setDragPosition({
      x: event.clientX - offset.x,
      y: event.clientY - offset.y,
    });
  };

  const handlePointerUp = () => {
    dragOffsetRef.current = null;
    setIsDragging(false);
  };

  useEffect(() => {
    if (!isDragging) {
      return;
    }
    document.addEventListener('pointermove', handlePointerMove);
    document.addEventListener('pointerup', handlePointerUp);
    return () => {
      document.removeEventListener('pointermove', handlePointerMove);
      document.removeEventListener('pointerup', handlePointerUp);
    };
  }, [isDragging]);

  const handleStartDrag = (event: JSX.TargetedEvent<HTMLHeadingElement, PointerEvent>) => {
    event.preventDefault();
    const rect = dialogRef.current?.getBoundingClientRect();
    if (!rect) {
      return;
    }
    const currentX = dragPosition?.x ?? rect.left;
    const currentY = dragPosition?.y ?? rect.top;
    dragOffsetRef.current = {
      x: event.clientX - currentX,
      y: event.clientY - currentY,
    };
    setIsDragging(true);
    event.currentTarget.setPointerCapture?.(event.pointerId);
  };

  const dialogStyle = {
    position: 'absolute' as const,
    left: dragPosition ? `${dragPosition.x}px` : '50%',
    top: dragPosition ? `${dragPosition.y}px` : '50%',
    transform: dragPosition ? 'translate(0, 0)' : 'translate(-50%, -50%)',
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
      <div
        ref={dialogRef}
        className="w-full max-w-3xl rounded-lg border border-[#1f2230] bg-[#0b0d11] shadow-2xl text-white"
        style={dialogStyle}
      >
        <div className={`flex items-center justify-between border-b border-[#1f2230] px-4 py-3 ${isDragging ? 'cursor-grabbing' : 'cursor-grab'}`}  onPointerDown={handleStartDrag}>
          <h3 className="text-lg font-semibold">
            Enviar emails
          </h3>
        </div>
        <div className="space-y-4 px-4 py-4">
          {loading ? (
            <p className="text-sm text-white/70">Generando lista de destinatarios...</p>
          ) : (
            <>
              <div className="space-y-1">
                <label className="block text-xs font-semibold text-white/70">Asunto</label>
                <input
                  type="text"
                  value={subject}
                  onInput={(event) => onSubjectChange((event.currentTarget as HTMLInputElement).value)}
                  className="w-full rounded border border-[#1f2230] bg-[#15182c] px-3 py-2 text-sm text-white focus:border-brand-blue focus:outline-none"
                  disabled={sending}
                />
              </div>
              <div className="space-y-1">
                <label className="block text-xs font-semibold text-white/70">Mensaje</label>
                <textarea
                  value={body}
                  onInput={(event) => onBodyChange((event.currentTarget as HTMLTextAreaElement).value)}
                  rows={6}
                  className="w-full rounded border border-[#1f2230] bg-[#15182c] px-3 py-2 text-sm text-white focus:border-brand-blue focus:outline-none"
                  disabled={sending}
                />
              </div>
              <div className="space-y-1">
                <label className="block text-xs font-semibold text-white/70">Adjuntos</label>
                <input
                  type="file"
                  multiple
                  onInput={(event: JSX.TargetedEvent<HTMLInputElement, Event>) => {
                    onAddAttachments(event.currentTarget.files);
                    event.currentTarget.value = '';
                  }}
                  className="text-xs text-white cursor-pointer"
                  disabled={sending}
                />
                {attachments.length > 0 && (
                  <ul className="space-y-1 text-xs text-white/80">
                    {attachments.map((file, idx) => (
                      <li key={`${file.name}-${idx}`} className="flex items-center justify-between gap-2">
                        <span className="truncate">{file.name}</span>
                        <button
                          type="button"
                          onClick={() => onRemoveAttachment(idx)}
                          className="uppercase tracking-wider text-brand-pink cursor-pointer hover:text-red-700"
                          disabled={sending}
                        >
                          Eliminar
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
              <div className="space-y-2">
                <p className="text-xs font-semibold text-white/60">Destinatarios ({selectedCount})</p>
                {candidates.length === 0 ? (
                  <p className="text-xs text-white/60">No hay correos disponibles para los filtros actuales.</p>
                ) : (
                  <div className="max-h-48 space-y-1 overflow-y-auto rounded border border-[#1f2230] px-3 py-2">
                    {candidates.map((candidate, idx) => (
                      <label key={`${candidate.email}-${idx}`} className="flex items-center gap-2 text-sm cursor-pointer">
                        <input
                          type="checkbox"
                          checked={candidate.selected}
                          onInput={() => onToggleRecipient(idx)}
                          disabled={sending}
                          className="h-4 w-4 rounded border border-[#444] bg-[#15182c] text-brand-blue focus:outline-none focus:ring-2 focus:ring-brand-blue"
                        />
                        <span className="truncate">{candidate.email}</span>
                      </label>
                    ))}
                  </div>
                )}
              </div>
              {error && <p className="text-xs text-brand-pink">{error}</p>}
              <div className="flex justify-end gap-2">
                <button
                  type="button"
                  onClick={onClose}
                  className="rounded border border-[#323640] px-4 py-2 text-sm text-white/70 hover:text-red-600 cursor-pointer"
                  disabled={sending}
                >
                  Cancelar
                </button>
                <button
                  type="button"
                  onClick={onSend}
                  disabled={sending || selectedCount === 0 || body.trim() === ''}
                  className={`rounded px-4 py-2 text-sm  ${
                    sending || selectedCount === 0 || body.trim() === ''
                      ? 'bg-[#1c2230] text-white/60 cursor-not-allowed'
                      : 'bg-brand-pink text-white hover:bg-[#ff8b6d] cursor-pointer'
                  }`}
                >
                  {sending ? 'Enviando...' : 'Enviar emails'}
                </button>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
