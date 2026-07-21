export interface Finding {
  id: string;
  cluster: string;
  rule: string;
  severity: "error" | "warning" | "info";
  file: string;
  line: number;
  column: number;
  message: string;
  fixHint?: string;
  docUrl?: string;
}

export interface Summary {
  errors: number;
  warnings: number;
  info: number;
  total: number;
}

export interface KDoctorReport {
  schemaVersion: string;
  projectType: string;
  healthScore: number;
  summary: Summary;
  findings: Finding[];
}
