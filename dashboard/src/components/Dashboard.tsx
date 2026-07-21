import React from 'react';
import { ArrowLeft, Activity, AlertTriangle, Info, XCircle } from 'lucide-react';
import type { KDoctorReport } from '../types';
import ScoreTrend from './ScoreTrend';
import ClusterCharts from './ClusterCharts';
import FindingsTable from './FindingsTable';

interface DashboardProps {
  report: KDoctorReport;
  onReset: () => void;
}

const Dashboard: React.FC<DashboardProps> = ({ report, onReset }) => {
  return (
    <div className="min-h-screen bg-ink-950 text-gray-200">
      {/* Header */}
      <header className="sticky top-0 z-50 glass-panel border-b border-ink-800 px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <button 
            onClick={onReset}
            className="p-2 hover:bg-ink-800 rounded-lg transition-colors text-ink-400 hover:text-white"
            title="Upload another report"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <div>
            <h1 className="text-xl font-bold text-white flex items-center gap-2">
              <Activity className="w-6 h-6 text-glow-cyan" />
              kdoctor Report
            </h1>
            <p className="text-sm text-ink-400">
              Project Type: <span className="text-glow-cyan uppercase font-mono">{report.projectType}</span>
            </p>
          </div>
        </div>
        
        <div className="flex gap-4">
          <div className="flex items-center gap-2 px-3 py-1.5 bg-ink-900 rounded border border-ink-800">
            <XCircle className="w-4 h-4 text-glow-red" />
            <span className="font-mono">{report.summary.errors} Errors</span>
          </div>
          <div className="flex items-center gap-2 px-3 py-1.5 bg-ink-900 rounded border border-ink-800">
            <AlertTriangle className="w-4 h-4 text-glow-amber" />
            <span className="font-mono">{report.summary.warnings} Warnings</span>
          </div>
          <div className="flex items-center gap-2 px-3 py-1.5 bg-ink-900 rounded border border-ink-800">
            <Info className="w-4 h-4 text-glow-cyan" />
            <span className="font-mono">{report.summary.info} Info</span>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto p-6 space-y-6">
        
        {/* Top Row: Score & Charts */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-1">
            <ScoreTrend score={report.healthScore} />
          </div>
          <div className="lg:col-span-2">
            <ClusterCharts findings={report.findings} />
          </div>
        </div>

        {/* Findings Table */}
        <FindingsTable findings={report.findings} />

      </main>
    </div>
  );
};

export default Dashboard;
