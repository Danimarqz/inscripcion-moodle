import { useCallback, useState } from 'preact/hooks';

export function useAsyncTask() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const run = useCallback(
    async <T>(task: () => Promise<T>, options: { suppressLoading?: boolean } = {}) => {
      if (!options.suppressLoading) {
        setLoading(true);
      }
      setError(null);
      try {
        return await task();
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        setError(message);
        throw err;
      } finally {
        if (!options.suppressLoading) {
          setLoading(false);
        }
      }
    },
    [],
  );

  return { loading, error, run, setError, setLoading } as const;
}
