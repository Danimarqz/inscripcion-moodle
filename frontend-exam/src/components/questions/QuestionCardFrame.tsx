import type { ComponentChildren } from 'preact';

interface QuestionCardFrameProps {
  label: string;
  badgeText: string;
  badgeClassName: string;
  meta?: ComponentChildren;
  children: ComponentChildren;
}

export default function QuestionCardFrame({
  label,
  badgeText,
  badgeClassName,
  meta,
  children,
}: QuestionCardFrameProps) {
  return (
    <div className="bg-[#2a2d33] p-6 mb-4 mt-4 rounded-lg shadow-lg border border-transparent hover:border-brand-blue/40 transition-colors">
      <div className="flex items-center justify-between mb-4">
        <p className="font-bold text-lg text-brand-blue text-opacity-80">{label}</p>
        <span className={`text-xs font-semibold px-2 py-1 rounded ${badgeClassName}`}>{badgeText}</span>
      </div>
      {meta}
      {children}
    </div>
  );
}
