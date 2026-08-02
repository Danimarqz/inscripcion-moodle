import { useState, useRef, useEffect } from 'preact/hooks';
import { getQuestionFeedbackVideo, patchQuestionFeedbackVideo } from '../../services/adminService';

export default function FeedbackVideoKeyField({ questionId, token }: { questionId: number; token: string }) {
  const [key, setKey] = useState('');
  const [status, setStatus] = useState<'idle' | 'saving' | 'ok' | 'error'>('idle');
  const rootRef = useRef<HTMLDivElement>(null);

  // Fetch the saved key only when this field scrolls into view, so opening an
  // exam with many questions doesn't fire one request per question at once.
  useEffect(() => {
    const el = rootRef.current;
    if (!el) return;
    let done = false;
    const load = () => {
      if (done) return;
      done = true;
      getQuestionFeedbackVideo(questionId, token)
        .then((k) => { if (k) setKey(k); })
        .catch(() => {});
    };
    const observer = new IntersectionObserver((entries) => {
      if (entries.some((e) => e.isIntersecting)) {
        load();
        observer.disconnect();
      }
    }, { rootMargin: '200px' });
    observer.observe(el);
    return () => observer.disconnect();
  }, [questionId, token]);

  async function save(value: string | null) {
    setStatus('saving');
    try {
      await patchQuestionFeedbackVideo(questionId, value, token);
      setStatus('ok');
      if (value === null) setKey('');
      setTimeout(() => setStatus('idle'), 2000);
    } catch {
      setStatus('error');
      setTimeout(() => setStatus('idle'), 3000);
    }
  }

  return (
    <div ref={rootRef} className="mt-4 pt-3 border-t border-[#333]">
      <p className="text-xs font-semibold text-gray-400 mb-1">Vídeo de feedback (key S3)</p>
      <div className="flex gap-2 items-center flex-wrap">
        <input
          type="text"
          value={key}
          placeholder="examen-2025/pregunta-42"
          onInput={(e) => setKey((e.target as HTMLInputElement).value.trim())}
          className="flex-1 min-w-0 px-3 py-1.5 rounded border border-[#444] bg-[#1f2229] text-white text-sm focus:outline-none focus:border-brand-blue"
        />
        <button
          type="button"
          disabled={status === 'saving' || !key}
          onClick={() => save(key)}
          className="px-3 py-1.5 rounded bg-brand-blue text-white text-sm font-semibold disabled:opacity-50 cursor-pointer"
        >
          {status === 'saving' ? '…' : 'Guardar'}
        </button>
        <button
          type="button"
          disabled={status === 'saving'}
          onClick={() => save(null)}
          className="px-3 py-1.5 rounded border border-red-500/50 text-red-400 text-sm font-semibold disabled:opacity-50 cursor-pointer"
        >
          Borrar
        </button>
        {status === 'ok' && <span className="text-xs text-green-400">✓ Guardado</span>}
        {status === 'error' && <span className="text-xs text-red-400">Error al guardar</span>}
      </div>
    </div>
  );
}

