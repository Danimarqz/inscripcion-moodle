import { useEffect, useRef } from 'preact/hooks';
import {
  Chart,
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  TimeScale,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js';
import 'chartjs-adapter-date-fns';
import type { DayBucket } from '../types/logs';

Chart.register(
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  TimeScale,
  Tooltip,
  Legend,
  Filler,
);

interface Props {
  data: DayBucket[];
}

export default function LogsHitsChart({ data }: Props) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const chartRef = useRef<Chart | null>(null);

  useEffect(() => {
    if (!canvasRef.current) return;
    const points = data.map((d) => ({ x: new Date(d.date).getTime(), y: d.hits }));

    if (chartRef.current) {
      chartRef.current.data.datasets[0].data = points;
      chartRef.current.update();
      return;
    }

    chartRef.current = new Chart(canvasRef.current, {
      type: 'line',
      data: {
        datasets: [
          {
            label: 'Visitas / día',
            data: points,
            borderColor: '#ec4899',
            backgroundColor: 'rgba(236, 72, 153, 0.15)',
            fill: true,
            tension: 0.25,
            pointRadius: 2,
          },
        ],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          x: {
            type: 'time',
            time: { unit: 'day', tooltipFormat: 'yyyy-MM-dd' },
            ticks: { color: '#9ca3af' },
            grid: { color: 'rgba(255,255,255,0.05)' },
          },
          y: {
            beginAtZero: true,
            ticks: { color: '#9ca3af' },
            grid: { color: 'rgba(255,255,255,0.05)' },
          },
        },
        plugins: {
          legend: { labels: { color: '#e5e7eb' } },
          tooltip: { mode: 'index', intersect: false },
        },
      },
    });
  }, [data]);

  useEffect(() => {
    return () => {
      chartRef.current?.destroy();
      chartRef.current = null;
    };
  }, []);

  return (
    <div class="bg-[#1a1c22] rounded-lg p-4 h-80">
      <canvas ref={canvasRef} />
    </div>
  );
}
