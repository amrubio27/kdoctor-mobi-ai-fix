import React, { useState, useCallback } from 'react';
import { UploadCloud, FileJson } from 'lucide-react';
import type { KDoctorReport } from './types';
import Dashboard from './components/Dashboard';

function App() {
  const [report, setReport] = useState<KDoctorReport | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isDragging, setIsDragging] = useState(false);

  const handleFileUpload = (file: File) => {
    const reader = new FileReader();
    reader.onload = (e) => {
      try {
        const json = JSON.parse(e.target?.result as string);
        if (json.schemaVersion && json.healthScore !== undefined) {
          setReport(json as KDoctorReport);
          setError(null);
        } else {
          setError('Invalid file format. Please upload a valid kdoctor scan JSON report.');
        }
      } catch (err) {
        setError('Failed to parse JSON file.');
      }
    };
    reader.readAsText(file);
  };

  const onDrop = useCallback((e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragging(false);
    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      handleFileUpload(e.dataTransfer.files[0]);
    }
  }, []);

  const onDragOver = useCallback((e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragging(true);
  }, []);

  const onDragLeave = useCallback((e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragging(false);
  }, []);

  if (report) {
    return <Dashboard report={report} onReset={() => setReport(null)} />;
  }

  return (
    <div className="min-h-screen bg-ink-950 flex flex-col items-center justify-center p-6 text-gray-200">
      <div className="max-w-2xl w-full text-center mb-12">
        <h1 className="text-4xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-glow-cyan to-glow-green mb-4">
          kdoctor Dashboard
        </h1>
        <p className="text-ink-400 text-lg">
          Upload your <code className="bg-ink-800 px-2 py-1 rounded text-glow-cyan">kdoctor scan --json</code> output to view the Health Score, insights, and findings.
        </p>
      </div>

      <div
        onDrop={onDrop}
        onDragOver={onDragOver}
        onDragLeave={onDragLeave}
        className={`w-full max-w-xl p-12 border-2 border-dashed rounded-2xl transition-all duration-200 flex flex-col items-center justify-center cursor-pointer ${
          isDragging
            ? 'border-glow-cyan bg-glow-cyan/5 shadow-[0_0_30px_rgba(33,230,255,0.15)]'
            : 'border-ink-700 bg-ink-900/50 hover:border-glow-green/50 hover:bg-ink-800'
        }`}
        onClick={() => document.getElementById('file-upload')?.click()}
      >
        <UploadCloud className={`w-16 h-16 mb-6 ${isDragging ? 'text-glow-cyan' : 'text-ink-400'}`} />
        <h3 className="text-xl font-semibold mb-2">Drag & Drop your JSON report here</h3>
        <p className="text-ink-500 mb-6">or click to browse from your computer</p>
        
        <input
          id="file-upload"
          type="file"
          accept=".json,application/json"
          className="hidden"
          onChange={(e) => {
            if (e.target.files && e.target.files.length > 0) {
              handleFileUpload(e.target.files[0]);
            }
          }}
        />
        
        <button className="flex items-center gap-2 bg-ink-800 hover:bg-ink-700 text-white px-6 py-3 rounded-lg font-medium transition-colors border border-ink-600">
          <FileJson className="w-5 h-5" />
          Select JSON File
        </button>
      </div>

      {error && (
        <div className="mt-6 p-4 bg-glow-red/10 border border-glow-red/30 rounded-lg text-glow-red flex items-center gap-3">
          <span className="font-semibold">Error:</span> {error}
        </div>
      )}
    </div>
  );
}

export default App;
