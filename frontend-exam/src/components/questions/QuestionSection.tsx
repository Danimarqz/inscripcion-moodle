import type { ComponentChildren, JSX } from 'preact';

interface QuestionSectionProps<T> {
  title: string;
  entries: T[];
  description?: ComponentChildren;
  emptyMessage?: string;
  emptyMessageClassName?: string;
  renderEntry: (entry: T, position: number) => ComponentChildren;
  showCount?: boolean;
  titleTag?: keyof JSX.IntrinsicElements;
  titleClassName?: string;
  sectionClassName?: string;
}

export default function QuestionSection<T>({
  title,
  entries,
  description,
  emptyMessage,
  emptyMessageClassName,
  renderEntry,
  showCount = true,
  titleTag = 'h3',
  titleClassName,
  sectionClassName = 'mt-6',
}: QuestionSectionProps<T>) {
  const TitleTag = titleTag as keyof JSX.IntrinsicElements;
  const headingClasses = titleClassName ?? 'text-lg font-semibold text-brand-pink mb-2';
  return (
    <section className={sectionClassName}>
      <TitleTag className={headingClasses}>
        {title}
        {showCount && <> ({entries.length})</>}
      </TitleTag>
      {description && <div className="text-xs text-gray-400 mb-4">{description}</div>}
      {entries.length === 0 ? (
        emptyMessage ? (
          <p className={`${emptyMessageClassName ?? 'text-sm text-red-400 bg-red-500/10 border border-red-500/40 p-3 rounded'}`}>
            {emptyMessage}
          </p>
        ) : null
      ) : (
        entries.map((entry, index) => renderEntry(entry, index + 1))
      )}
    </section>
  );
}
