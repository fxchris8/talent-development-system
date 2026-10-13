export type PageType = "prev" | "next";
export type FilterType = "" | "MDP" | "FDP" | "SDP";

export interface IReport {
  id: number;
  seamanCode: string;
  seafarerCode: string;
  nama: string;
  jabatan: string;
  vesselName: string;
  certificate: string;
  age: string;
  tanggalLahir: string;
  startDate: string;
  user?: string;

  // Catatan Indisipliner
  warningLetter: string;
  caseHistory: string;
  yearOfCase: string;

  // Data History Vessel
  vesselHistory: string;

  // Kinerja
  konditeReview: number;
  kpiVessel: number;
  performanceScore: number;
  valueAssessment: number;
  valueAssessmentScore: number;
  assessmentCenter: number;
  potentialScore: number;
  havQuadran: number;
  havMapping: string;
  competencyGapAnalysis: string;
  totalGap: number;
  strength: number;
  talentClassified: string;
  idpProgram: string;
  trainingStatus?: string; // Ongoing | Completed | Exited
  exitedProgram?: string; // program yang ditinggalkan saat naik jabatan
  idp: string; //  tambahan
  havQuadran2: number;
  talentClassified2: string;
  readiness: string;
  totalReadinessUpdateMonths: number;
  certificateEligible: string;

  // Training, Mentoring, Coaching
  trainingCompleted: string;
  trainingPlanned: string;
  mentoringCompleted: string;
  mentoringPlanned: string;
  coachingCompleted: string;
  coachingPlanned: string;

  // Succession Plan
  dataIncumbent: string;
  successionVessel: string;
  successionRank: string;

  // Individual Development Plan
  idpStart: string;
  idpMentor: string;
  idpCoach: string;

  // Mentoring Report Link
  mentoringReportId?: number | null;
  mentoringReport?: IMentoringReport;

  // Assessment Type Scores
  reportScores?: Array<{
    score?: number;
    assessmentType?: {
      assessmentTypeName?: string;
    };
  }>;

  competencies: IGapCompetency[];
  batchId?: number | null;
}

export interface IMentoringReport {
  id: number;
  mentorName: string;
  period: string;
  menteeNames: string[];
  department: string;
  program: string;
  programTitle: string;
  sessionNumber: string;
  date: string;
  duration: string;
  purpose: string;
  observation: string;
  reflection: string;
  actionPlan: string;
  additionalNotes: string;
  reportIds: number[];
  createdAt: string;
  updatedAt: string;
}

export interface IPaginationData<T> {
  results: T[];
  first_id: number | null;
  last_id: number | null;
  page_size: number;
  has_more: boolean;
  first_page: boolean;
  total?: number;
}

export interface IPaginationRequest {
  anchorId: number | null;
  page: PageType;
  pageSize: number;
  filter: FilterType;
  batchId?: number | null;
  query?: string;
}

export interface IUser {
  id?: number;
  nama?: string;
  jabatan?: string;
}

export interface IAssessment {
  // PascalCase (from backend)
  assessmentId?: number;
  role?: string;
  assessmentName?: string;
  usingTimer?: boolean;
  timerLimitMinutes?: number | null;

  // camelCase (for compatibility)
  id?: number;
  name?: string;
  using_timer?: boolean;
  timer_limit_minutes?: number | null;
}

export interface IUser {
  // PascalCase (from backend)
  ID?: number;
  Nama?: string;
  Jabatan?: string;
  SeamanCode?: string;
  SeafarerCode?: string;
  VesselName?: string;
  Certificate?: string;
  Age?: string;
  TanggalLahir?: string;
  StartDate?: string;
  WarningLetter?: string;
  CaseHistory?: string;
  YearOfCase?: string;
  VesselHistory?: string;
  KonditeReview?: number;
  KpiVessel?: number;
  PerformanceScore?: number;
  ValueAssessment?: number;
  AssessmentCenter?: number;
  PotentialScore?: number;
  HavQuadran?: number;
  HavMapping?: string;
  CompetencyGapAnalysis?: string;
  TotalGap?: number;
  Strength?: number;
  TalentClassified?: string;
  IDPProgram?: string;
  HavQuadran2?: number;
  TalentClassified2?: string;
  Readiness?: string;
  CertificateEligible?: string;
  TrainingCompleted?: string;
  TrainingPlanned?: string;
  MentoringCompleted?: string;
  MentoringPlanned?: string;
  CoachingCompleted?: string;
  CoachingPlanned?: string;
  DataIncumbent?: string;
  SuccessionVessel?: string;
  SuccessionRank?: string;
  IDPStart?: string;
  IDPMentor?: string;
  IDPCoach?: string;
  ReadinessMonth?: number | null;
  EducationFulfillmentMonths?: number | null;
  TotalReadinessUpdateMonths?: number | null;
  Keterangan?: string | null;
  TmNm?: string | null;
  CreatedAt?: string;
  UpdatedAt?: string;

  // camelCase (for compatibility)
  id?: number;
  nama?: string;
  jabatan?: string;
  seamanCode?: string;
  seafarerCode?: string;
  vesselName?: string;
  certificate?: string;
  age?: string;
  created_at?: string;
  updated_at?: string;
}

export interface IAssignmentFlat {
  id: number;
  seafarerCode: string;
  nama: string;
  assessmentTypeId: number;
  assessmentType: string;
  attempts: number;
  status: string;
  batchId?: number | null;
}

// === For Create / Update API ===
export interface IAssignmentCreate {
  seafarerCode: string;
  assessmentTypeId: number;
  batchId?: number | null;
  status?: string;
  createdBy: string;
}

export interface IAssignmentUpdate {
  id: number;
  assessmentTypeId?: number;
  status?: string;
}

export interface ICompetencyType {
  id: number;
  code: string;
  name: string;
  description?: string;
  category?: string;
  isActive?: boolean;
}

// models/seaman.ts
export interface SeamanLookup {
  name: string;
  seamanCode: string;
  alreadyAdded?: boolean;
}

export interface IGapCompetency {
  id?: number;
  competencyTypeId?: number;
  competencyType?: {
    id: number;
    code: string;
    name: string; // ✓ Must include name
  };
}

// Re-export report types from specific types module
export type {
  ReportData,
  ApiResponse,
  ReportSection,
  SectionAnswer,
  PersonalInfo,
  ApiError,
  AssessmentAnswer,
} from "./report-types";
