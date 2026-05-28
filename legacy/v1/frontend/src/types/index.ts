export type StageCode =
  | 'submit' | 'review' | 'propose'
  | 'meeting' | 'arbitrate' | 'consult'
  | 'implement' | 'evaluate'

export type BranchCode = 'dispute' | 'consensus' | null

export type StatusCode =
  | 'pending' | 'processing' | 'meeting' | 'arbitrate' | 'consulting' | 'done' | 'overdue'

export type PriorityCode = 'critical' | 'high' | 'normal' | 'low'

export interface StageDef {
  code: StageCode
  label: string
  branch: 'dispute' | 'consensus' | null
  seq: number
}

export interface User {
  id: number
  name: string
  dept: string
  avatarColor?: string
}

export interface Problem {
  id: string
  title: string
  description: string
  category: string
  priority: PriorityCode
  status: StatusCode
  branch: BranchCode
  currentStage: StageCode
  submitterId: number
  submitterDept: string
  handlerName: string
  handlerDept: string
  submitDate: string
  dueDate: string
  progress: number
  overdue: boolean
  overdueDays: number
  latest: string
  tags: string[]
  participants: string[]
}

export interface StageHistory {
  id: number
  problemId: string
  stage: StageCode
  occurredAt: string
  actorUserId: number
  actorName: string
  actorDept: string
  note: string
  files: string[]
  branchChoice: string | null
}

export interface Measure {
  id: number
  problemId: string
  code: string
  title: string
  owner: string
  status: 'proposed' | 'drafting' | 'approved' | 'in_progress' | 'completed'
  hasDispute: boolean
  progress: number
  displayOrder: number
}

export interface Dispute {
  id: number
  problemId: string
  point: string
  resolution: string | null
  displayOrder: number
}

export interface DisputePosition {
  id: number
  disputeId: number
  party: string
  view: string
}

export interface DisputeWithPositions {
  dispute: Dispute
  positions: DisputePosition[]
}

export interface Message {
  id: number
  problemId: string
  actorUserId: number
  actorName: string
  content: string
  mentions: string[]
  occurredAt: string
}

export interface Attachment {
  id: number
  problemId: string
  stage: StageCode
  fileName: string
  fileSize: number
  contentType: string
  objectKey: string
  uploaderName: string
  uploadedAt: string
}

export interface ConsultStat {
  problemId: string
  totalCount: number
  supportCount: number
  neutralCount: number
  opposeCount: number
  startDate: string
  endDate: string
  brief: string
  revision: string
}

export interface Evaluation {
  id: number
  problemId: string
  evaluatorName: string
  party: string
  quality: number
  speed: number
  collab: number
  satisfaction: number
  overall: number
  comment: string
  archiveBestPractice: boolean
}

export interface ProblemDetail {
  problem: Problem
  submitter: User
  history: StageHistory[]
  measures: Measure[]
  disputes: DisputeWithPositions[]
  messages: Message[]
  attachments: Attachment[]
  consult: ConsultStat | null
  evaluations: Evaluation[]
}

export interface PageResult<T> {
  items: T[]
  total: number
  page: number
  size: number
}

export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

export interface DashboardOverview {
  kpis: {
    total: number
    pendingReview: number
    pendingAssign: number
    processing: number
    done: number
    overdue: number
  }
  processingBreakdown: { k: string; v: number }[]
  categories: { k: string; v: number }[]
  topSubmitterDepts: { k: string; v: number }[]
  topHandlerDepts: { k: string; v: number }[]
  overdueByDept: { k: string; v: number }[]
  trend: { month: string; submit: number; done: number }[]
  satisfaction: { name: string; score: number; evaluations: number }[]
  disputeStats: {
    totalPropose: number
    withDispute: number
    consultPath: number
    arbitrateDone: number
    disputeRate: number
    avgMeetings: number
  }
}

export interface NotificationDigest {
  mentions: { type: string; problemId: string; text: string; time: string }[]
  overdues: { type: string; problemId: string; text: string; time: string }[]
  dueSoon: { type: string; problemId: string; text: string; time: string }[]
}
