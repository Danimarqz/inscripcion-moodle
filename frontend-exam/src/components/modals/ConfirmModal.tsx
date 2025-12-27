import { useEffect, useRef, useState } from 'preact/hooks';
import type { JSX } from 'preact';

interface ConfirmModalProps {
  isOpen: boolean;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  isDanger?: boolean;
  onConfirm: () => void;
  onClose: () => void;
}

export default function ConfirmModal({
  isOpen,
  title,
  message,
  confirmText = 'Confirmar',
  cancelText = 'Cancelar',
  isDanger = false,
  onConfirm,
  onClose,
}: ConfirmModalProps) {
  const [dragPosition, setDragPosition] = useState<{ x: number; y: number } | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const dialogRef = useRef<HTMLDivElement>(null);
  const dragOffsetRef = useRef<{ x: number; y: number } | null>(null);

  const handlePointerMove = (event: PointerEvent) => {
    const offset = dragOffsetRef.current;
    if (!offset) return;
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
    if (!isDragging) return;
    document.addEventListener('pointermove', handlePointerMove);
    document.addEventListener('pointerup', handlePointerUp);
    return () => {
      document.removeEventListener('pointermove', handlePointerMove);
      document.removeEventListener('pointerup', handlePointerUp);
    };
  }, [isDragging]);

  const handleStartDrag = (event: JSX.TargetedPointerEvent<HTMLHeadingElement>) => {
    event.preventDefault();
    const rect = dialogRef.current?.getBoundingClientRect();
    if (!rect) return;
    const currentX = dragPosition?.x ?? rect.left;
    const currentY = dragPosition?.y ?? rect.top;
    dragOffsetRef.current = {
      x: event.clientX - currentX,
      y: event.clientY - currentY,
    };
    setIsDragging(true);
    (event.currentTarget as HTMLElement).setPointerCapture?.(event.pointerId);
  };

  if (!isOpen) return null;

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
            {title}
          </h3>
        </div>
        
        <div className="p-6 space-y-4">
          <p className="text-sm text-gray-300 leading-relaxed">
            {message}
          </p>
          
          <div className="flex justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-xs font-semibold uppercase tracking-wider rounded border border-[#323640] text-white/70 hover:bg-[#1f2229] transition-colors"
            >
              {cancelText}
            </button>
            <button
              type="button"
              onClick={() => {
                onConfirm();
                onClose();
              }}
              className={`px-4 py-2 text-xs font-semibold uppercase tracking-wider rounded transition-colors ${
                isDanger 
                  ? 'bg-brand-pink text-white hover:bg-[#ff8b6d]' 
                  : 'bg-brand-blue text-white hover:bg-[#12b2d4]'
              }`}
            >
              {confirmText}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
