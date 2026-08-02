import type { JSX } from 'preact';

interface ExamResultTypeSelectorProps {
  value: string;
  onChange: (value: string) => void;
}

const RESULT_TYPES = ['General', 'Promoción interna', 'Discapacidad', 'Otros'];

export default function ExamResultTypeSelector({ value, onChange }: ExamResultTypeSelectorProps) {
  return (
    <div className="mt-6 mb-6">
      <label className="block text-sm font-medium text-gray-300 mb-2">
        Tipo de convocatoria
      </label>
      <select
        value={value}
        onChange={(e: JSX.TargetedEvent<HTMLSelectElement>) => onChange((e.target as HTMLSelectElement).value)}
        className="w-full px-4 py-3 rounded-lg bg-[#2a2d33] border border-[#555] text-white focus:ring-2 focus:ring-brand-pink focus:border-transparent transition-all"
      >
        {RESULT_TYPES.map((t) => (
          <option key={t} value={t}>{t}</option>
        ))}
      </select>
    </div>
  );
}
