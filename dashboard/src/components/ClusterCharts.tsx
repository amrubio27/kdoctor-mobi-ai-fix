import React, { useMemo } from 'react';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';
import type { Finding } from '../types';

interface ClusterChartsProps {
  findings: Finding[];
}

const COLORS = {
  error: '#ff5470',   // glow-red
  warning: '#ffb84a', // glow-amber
  info: '#21e6ff'     // glow-cyan
};

const ClusterCharts: React.FC<ClusterChartsProps> = ({ findings }) => {
  const data = useMemo(() => {
    const counts: Record<string, { cluster: string; error: number; warning: number; info: number; total: number }> = {};
    
    findings.forEach(f => {
      if (!counts[f.cluster]) {
        counts[f.cluster] = { cluster: f.cluster, error: 0, warning: 0, info: 0, total: 0 };
      }
      counts[f.cluster][f.severity]++;
      counts[f.cluster].total++;
    });

    return Object.values(counts).sort((a, b) => b.total - a.total);
  }, [findings]);

  if (data.length === 0) {
    return (
      <div className="h-full bg-ink-900 border border-ink-800 rounded-2xl p-6 flex items-center justify-center">
        <p className="text-ink-400">No findings to display.</p>
      </div>
    );
  }

  return (
    <div className="h-full bg-ink-900 border border-ink-800 rounded-2xl p-6 flex flex-col">
      <h3 className="text-lg font-semibold text-white mb-6">Issues by Cluster</h3>
      
      <div className="flex-1 min-h-[300px]">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} layout="vertical" margin={{ top: 0, right: 30, left: 40, bottom: 0 }}>
            <XAxis type="number" stroke="#252d4a" tick={{ fill: '#6b7280' }} />
            <YAxis 
              dataKey="cluster" 
              type="category" 
              stroke="#252d4a" 
              tick={{ fill: '#d6dcec', fontSize: 12 }} 
              width={140}
            />
            <Tooltip 
              cursor={{ fill: 'rgba(255,255,255,0.02)' }}
              contentStyle={{ backgroundColor: '#101626', border: '1px solid #252d4a', borderRadius: '8px', color: '#fff' }}
              itemStyle={{ color: '#fff' }}
            />
            <Bar dataKey="error" stackId="a" fill={COLORS.error} name="Errors" radius={[0,0,0,0]} barSize={24} />
            <Bar dataKey="warning" stackId="a" fill={COLORS.warning} name="Warnings" radius={[0,0,0,0]} />
            <Bar dataKey="info" stackId="a" fill={COLORS.info} name="Info" radius={[0,4,4,0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};

export default ClusterCharts;
