import { useEffect, useRef, useState } from 'preact/hooks';
import Hls from 'hls.js';

interface Props {
  questionId: number;
  email: string;
  dni: string;
  examId: number;
}

export default function FeedbackVideoPlayer({ questionId, email, dni, examId }: Props) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const hlsRef = useRef<Hls | null>(null);
  const [error, setError] = useState<string | null>(null);
  const retriedRef = useRef(false);

  const apiBase = import.meta.env.PUBLIC_API_URL as string;
  const src = `${apiBase}/api/questions/${questionId}/feedback.m3u8?email=${encodeURIComponent(email)}&dni=${encodeURIComponent(dni)}&exam_id=${examId}`;

  function mount(video: HTMLVideoElement) {
    if (Hls.isSupported()) {
      const hls = new Hls();
      hlsRef.current = hls;
      hls.loadSource(src);
      hls.attachMedia(video);
      hls.on(Hls.Events.ERROR, (_evt, data) => {
        if (data.fatal) {
          hls.destroy();
          hlsRef.current = null;
          if (!retriedRef.current) {
            retriedRef.current = true;
            setTimeout(() => mount(video), 1000);
          } else {
            setError('No se pudo cargar el vídeo. Inténtalo de nuevo más tarde.');
          }
        }
      });
    } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
      // Safari native HLS
      video.src = src;
    } else {
      setError('Tu navegador no soporta la reproducción de este vídeo.');
    }
  }

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;
    retriedRef.current = false;
    setError(null);
    mount(video);
    return () => {
      hlsRef.current?.destroy();
      hlsRef.current = null;
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [src]);

  if (error) {
    return (
      <div className="flex w-full justify-center py-4">
        <p className="text-sm text-red-400">{error}</p>
      </div>
    );
  }

  return (
    <div className="flex w-full justify-center">
      <video
        ref={videoRef}
        controls
        playsInline
        preload="metadata"
        controlsList="nodownload noremoteplayback"
        disablePictureInPicture
        onContextMenu={(e) => e.preventDefault()}
        className="w-full max-w-[min(42rem,90vw)] aspect-video rounded-lg bg-black object-contain shadow-lg"
      />
    </div>
  );
}
