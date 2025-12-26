import type { SortableHeaderProps } from '../../types/exam';

export default function SortableHeader({
  label,
  sortKey,
  activeKey,
  direction,
  onSort,
}: SortableHeaderProps) {
  const isActive = activeKey === sortKey;
  const arrow = isActive ? (direction === 'asc' ? '↑' : '↓') : '↕';

  function handleClick() {
    onSort(sortKey);
  }

  return (
    <th className="px-3 py-2">
      <button
        type="button"
        onClick={handleClick}
        className={`flex items-center gap-1 text-xs font-semibold tracking-wide transition-colors ${
          isActive ? 'text-brand-pink' : 'text-gray-300 hover:text-brand-yellow'
        }`}
      >
        <span>{label}</span>
        <span className="text-xs">{arrow}</span>
      </button>
    </th>
  );
}
