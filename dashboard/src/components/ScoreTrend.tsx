import React from 'react';
import { ShieldAlert, ShieldCheck, Shield } from 'lucide-react';

interface ScoreTrendProps {
  score: number;
}

const ScoreTrend: React.FC<ScoreTrendProps> = ({ score }) => {
  let colorClass = "text-glow-green";
  let bgClass = "bg-glow-green/10";
  let borderClass = "border-glow-green/30";
  let shadowClass = "shadow-[0_0_40px_rgba(34,255,136,0.15)]";
  let Icon = ShieldCheck;
  let status = "Excellent";

  if (score < 50) {
    colorClass = "text-glow-red";
    bgClass = "bg-glow-red/10";
    borderClass = "border-glow-red/30";
    shadowClass = "shadow-[0_0_40px_rgba(255,84,112,0.15)]";
    Icon = ShieldAlert;
    status = "Critical";
  } else if (score < 80) {
    colorClass = "text-glow-amber";
    bgClass = "bg-glow-amber/10";
    borderClass = "border-glow-amber/30";
    shadowClass = "shadow-[0_0_40px_rgba(255,184,74,0.15)]";
    Icon = Shield;
    status = "Needs Improvement";
  }

  return (
    <div className={`h-full p-8 rounded-2xl border flex flex-col items-center justify-center relative overflow-hidden ${bgClass} ${borderClass} ${shadowClass}`}>
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center,_var(--tw-gradient-stops))] from-white/5 to-transparent opacity-50" />
      
      <h2 className="text-lg font-medium text-ink-400 mb-6 uppercase tracking-wider relative z-10">
        Health Score
      </h2>
      
      <div className="relative z-10 flex items-center justify-center mb-4">
        {/* Outer Glow Ring */}
        <div className={`absolute inset-0 rounded-full border-4 ${borderClass} animate-pulse-glow`} />
        
        {/* Score display */}
        <div className={`w-40 h-40 rounded-full flex items-center justify-center border-8 border-ink-900 bg-ink-950 shadow-2xl`}>
          <span className={`text-6xl font-black font-mono tracking-tighter ${colorClass}`}>
            {score}
          </span>
        </div>
      </div>

      <div className={`flex items-center gap-2 mt-4 px-4 py-2 rounded-full bg-ink-950 border border-ink-800 relative z-10`}>
        <Icon className={`w-5 h-5 ${colorClass}`} />
        <span className="font-semibold">{status}</span>
      </div>
    </div>
  );
};

export default ScoreTrend;
