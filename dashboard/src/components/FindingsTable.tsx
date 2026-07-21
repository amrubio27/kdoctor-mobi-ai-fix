import React, { useState, useMemo } from 'react';
import { Search, ChevronDown, ChevronUp } from 'lucide-react';
import type { Finding } from '../types';

interface FindingsTableProps {
  findings: Finding[];
}

type SortField = 'severity' | 'cluster' | 'rule' | 'file';
type SortOrder = 'asc' | 'desc';

const SeverityBadge = ({ severity }: { severity: string }) => {
  if (severity === 'error') {
    return <span className="px-2 py-1 text-xs font-medium rounded-md bg-glow-red/10 text-glow-red border border-glow-red/30">Error</span>;
  }
  if (severity === 'warning') {
    return <span className="px-2 py-1 text-xs font-medium rounded-md bg-glow-amber/10 text-glow-amber border border-glow-amber/30">Warning</span>;
  }
  return <span className="px-2 py-1 text-xs font-medium rounded-md bg-glow-cyan/10 text-glow-cyan border border-glow-cyan/30">Info</span>;
};

const FindingsTable: React.FC<FindingsTableProps> = ({ findings }) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [sortField, setSortField] = useState<SortField>('severity');
  const [sortOrder, setSortOrder] = useState<SortOrder>('asc');

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortOrder('asc');
    }
  };

  const filteredAndSorted = useMemo(() => {
    const severityWeight = { error: 0, warning: 1, info: 2 };
    
    let result = findings.filter(f => 
      f.message.toLowerCase().includes(searchTerm.toLowerCase()) ||
      f.file.toLowerCase().includes(searchTerm.toLowerCase()) ||
      f.rule.toLowerCase().includes(searchTerm.toLowerCase()) ||
      f.cluster.toLowerCase().includes(searchTerm.toLowerCase())
    );

    result.sort((a, b) => {
      let comparison = 0;
      if (sortField === 'severity') {
        comparison = severityWeight[a.severity] - severityWeight[b.severity];
      } else {
        comparison = a[sortField].localeCompare(b[sortField]);
      }
      return sortOrder === 'asc' ? comparison : -comparison;
    });

    return result;
  }, [findings, searchTerm, sortField, sortOrder]);

  return (
    <div className="bg-ink-900 border border-ink-800 rounded-2xl overflow-hidden flex flex-col">
      {/* Toolbar */}
      <div className="p-4 border-b border-ink-800 flex items-center justify-between bg-ink-950/50">
        <h3 className="text-lg font-semibold text-white">Findings ({filteredAndSorted.length})</h3>
        
        <div className="relative">
          <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-ink-500" />
          <input 
            type="text" 
            placeholder="Search findings..." 
            value={searchTerm}
            onChange={e => setSearchTerm(e.target.value)}
            className="bg-ink-950 border border-ink-800 rounded-lg pl-9 pr-4 py-2 text-sm text-gray-200 focus:outline-none focus:border-glow-cyan/50 focus:ring-1 focus:ring-glow-cyan/50 w-64"
          />
        </div>
      </div>

      {/* Table */}
      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm whitespace-nowrap">
          <thead className="bg-ink-950/80 text-ink-400 border-b border-ink-800 uppercase text-xs font-semibold">
            <tr>
              <th className="px-4 py-3 cursor-pointer hover:text-white transition-colors" onClick={() => handleSort('severity')}>
                <div className="flex items-center gap-1">Severity {sortField === 'severity' && (sortOrder === 'asc' ? <ChevronUp className="w-3 h-3"/> : <ChevronDown className="w-3 h-3"/>)}</div>
              </th>
              <th className="px-4 py-3 cursor-pointer hover:text-white transition-colors" onClick={() => handleSort('rule')}>
                <div className="flex items-center gap-1">Rule {sortField === 'rule' && (sortOrder === 'asc' ? <ChevronUp className="w-3 h-3"/> : <ChevronDown className="w-3 h-3"/>)}</div>
              </th>
              <th className="px-4 py-3 cursor-pointer hover:text-white transition-colors" onClick={() => handleSort('cluster')}>
                <div className="flex items-center gap-1">Cluster {sortField === 'cluster' && (sortOrder === 'asc' ? <ChevronUp className="w-3 h-3"/> : <ChevronDown className="w-3 h-3"/>)}</div>
              </th>
              <th className="px-4 py-3 cursor-pointer hover:text-white transition-colors" onClick={() => handleSort('file')}>
                <div className="flex items-center gap-1">Location {sortField === 'file' && (sortOrder === 'asc' ? <ChevronUp className="w-3 h-3"/> : <ChevronDown className="w-3 h-3"/>)}</div>
              </th>
              <th className="px-4 py-3 w-full">Message & Hint</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-ink-800/50">
            {filteredAndSorted.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-ink-500">No findings matched your search.</td>
              </tr>
            ) : (
              filteredAndSorted.map((f, i) => (
                <tr key={`${f.id}-${i}`} className="hover:bg-ink-800/30 transition-colors group">
                  <td className="px-4 py-3"><SeverityBadge severity={f.severity} /></td>
                  <td className="px-4 py-3 font-mono text-xs text-ink-300">{f.rule}</td>
                  <td className="px-4 py-3 text-ink-400">{f.cluster}</td>
                  <td className="px-4 py-3">
                    <div className="font-mono text-xs text-ink-300 max-w-[200px] truncate" title={`${f.file}:${f.line}`}>
                      {f.file.split('/').pop()}:{f.line}
                    </div>
                  </td>
                  <td className="px-4 py-3 whitespace-normal min-w-[300px]">
                    <div className="text-gray-300 mb-1">{f.message}</div>
                    {f.fixHint && (
                      <div className="text-xs text-glow-cyan/80 bg-glow-cyan/5 border border-glow-cyan/10 p-2 rounded flex gap-2">
                        <span className="font-semibold shrink-0">AI Hint:</span>
                        <span>{f.fixHint}</span>
                      </div>
                    )}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default FindingsTable;
