import { useEffect, useRef, useState } from 'preact/hooks';
import type { Exam } from '../../types/exam';

interface AssociateExamsModalProps {
  isOpen: boolean;
  exams: Exam[];
  selectedIds: number[];
  onSave: (ids: number[]) => void;
  onClose: () => void;
}

export default function AssociateExamsModal({
  isOpen,
  exams,
  selectedIds,
  onSave,
  onClose,
}: AssociateExamsModalProps) {
  const [draft, setDraft] = useState<number[]>(selectedIds);
  const [dragPosition, setDragPosition] = useState<{ x: number; y: number } | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const dialogRef = useRef<HTMLDivElement>(null);
  const dragOffsetRef = useRef<{ x: number; y: number } | null>(null);

  // Reset draft to the committed selection each time the modal opens.
  useEffect(() => {
    if (isOpen) setDraft(selectedIds);
  }, [isOpen, selectedIds]);

  const handlePointerMove = (event: PointerEvent) => {
    const offset = dragOffsetRef.current;
    if (!offset) return;
    setDragPosition({ x: event.clientX - offset.x, y: event.clientY - offset.y });
  };

  const handlePointerUp = () => {
    dragOffsetRef.current = null;
    setIsDragging(false);
  };

  useEffect(() => {
    if (!isDragging) return;
    document.addEventListener('pointermove', handlePointerMove);
    document.addEventListener('pointerup', handlePointerUp);
    return () => {
      document.removeEventListener('pointermove', handlePointerMove);
      document.removeEventListener('pointerup', handlePointerUp);
    };
  }, [isDragging]);

  const handleStartDrag = (event: PointerEvent) => {
    event.preventDefault();
    const rect = dialogRef.current?.getBoundingClientRect();
    if (!rect) return;
    const currentX = dragPosition?.x ?? rect.left;
    const currentY = dragPosition?.y ?? rect.top;
    dragOffsetRef.current = { x: event.clientX - currentX, y: event.clientY - currentY };
    setIsDragging(true);
    (event.currentTarget as HTMLElement).setPointerCapture?.(event.pointerId);
  };

  if (!isOpen) return null;

  const toggle = (id: number) =>
    setDraft((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));

  const dialogStyle = {
    position: 'absolute' as const,
    left: dragPosition ? `${dragPosition.x}px` : '50%',
    top: dragPosition ? `${dragPosition.y}px` : '50%',
    transform: dragPosition ? 'translate(0, 0)' : 'translate(-50%, -50%)',
  };

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 p-4">
      <div
        ref={dialogRef}
        className="w-full max-w-md rounded-lg border border-[#1f2230] bg-[#0b0d11] shadow-2xl text-white overflow-hidden"
        style={dialogStyle}
      >
        <div
          className={`flex items-center justify-between border-b border-[#1f2230] px-4 py-3 ${
            isDragging ? 'cursor-grabbing' : 'cursor-grab'
          }`}
          onPointerDown={handleStartDrag}
        >
          <h3 className="text-sm font-semibold tracking-wider uppercase text-white/90">
            Asociar exámenes para percentil
          </h3>
        </div>

        <div className="p-6 space-y-4">
          <p className="text-sm text-gray-400 leading-relaxed">
            El percentil se calcula sobre las notas de todos los exámenes seleccionados y este mismo (relación recíproca).
          </p>

          <div className="max-h-72 overflow-y-auto rounded border border-[#1f2230] divide-y divide-[#1f2230]">
            {exams.length === 0 ? (
              <p className="p-4 text-sm text-gray-500">No hay otros exámenes.</p>
            ) : (
              exams.map((e) => (
                <label
                  key={e.id}
                  className="flex items-center gap-3 px-4 py-2.5 cursor-pointer hover:bg-[#1f2229]"
                >
                  <input
                    type="checkbox"
                    checked={draft.includes(e.id)}
                    onChange={() => toggle(e.id)}
                    className="h-4 w-4 accent-brand-pink cursor-pointer"
                  />
                  <span className="text-sm">{e.name}</span>
                </label>
              ))
            )}
          </div>

          <div className="flex justify-between items-center pt-2">
            <span className="text-xs text-gray-500">{draft.length} seleccionado(s)</span>
            <div className="flex gap-3">
              <button
                type="button"
                onClick={onClose}
                className="px-4 py-2 text-xs font-semibold uppercase tracking-wider rounded border border-[#323640] text-white/70 hover:bg-[#1f2229] transition-colors"
              >
                Cancelar
              </button>
              <button
                type="button"
                onClick={() => {
                  onSave(draft);
                  onClose();
                }}
                className="px-4 py-2 text-xs font-semibold uppercase tracking-wider rounded bg-brand-blue text-white hover:bg-[#12b2d4] transition-colors"
              >
                Guardar
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
