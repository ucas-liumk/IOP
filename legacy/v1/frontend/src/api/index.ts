import http from './http'
import type {
  Problem, ProblemDetail, PageResult, User, DashboardOverview,
  Message, Attachment, NotificationDigest, StageDef,
} from '@/types'

// ============================ Users ============================
export const usersApi = {
  list: () => http.get<unknown, User[]>('/users'),
  me:   () => http.get<unknown, { id: number; name: string; dept: string }>('/users/me'),
}

// ============================ Stages metadata ============================
export const stagesApi = {
  all: () => http.get<unknown, StageDef[]>('/stages'),
}

// ============================ Problems ============================
export interface ProblemQuery {
  page?: number
  size?: number
  status?: string
  stage?: string
  priority?: string
  tab?: string
  query?: string
}

export const problemsApi = {
  list:   (q: ProblemQuery = {}) => http.get<unknown, PageResult<Problem>>('/problems', { params: q }),
  detail: (id: string) => http.get<unknown, ProblemDetail>(`/problems/${id}`),
  create: (req: any) => http.post<unknown, Problem>('/problems', req),

  review:    (id: string, req: any) => http.post<unknown, Problem>(`/problems/${id}/actions/review`, req),
  propose:   (id: string, req: any) => http.post<unknown, Problem>(`/problems/${id}/actions/propose`, req),
  meeting:   (id: string, req: any) => http.post<unknown, Problem>(`/problems/${id}/actions/meeting`, req),
  arbitrate: (id: string, req: any) => http.post<unknown, Problem>(`/problems/${id}/actions/arbitrate`, req),
  consult:   (id: string, req: any) => http.post<unknown, Problem>(`/problems/${id}/actions/consult`, req),
  implement: (id: string, req: any) => http.post<unknown, Problem>(`/problems/${id}/actions/implement`, req),
  evaluate:  (id: string, req: any) => http.post<unknown, Problem>(`/problems/${id}/actions/evaluate`, req),
}

// ============================ Dashboard ============================
export const dashboardApi = {
  overview: () => http.get<unknown, DashboardOverview>('/dashboard/overview'),
}

// ============================ Messages ============================
export const messagesApi = {
  list: (problemId: string) => http.get<unknown, Message[]>(`/messages/problem/${problemId}`),
  post: (problemId: string, content: string) =>
    http.post<unknown, Message>(`/messages/problem/${problemId}`, { content }),
}

// ============================ Files ============================
export const filesApi = {
  list: (problemId: string) => http.get<unknown, Attachment[]>(`/files/problem/${problemId}`),
  upload: (problemId: string, stage: string, file: File, onProgress?: (p: number) => void) => {
    const fd = new FormData()
    fd.append('problemId', problemId)
    fd.append('stage', stage)
    fd.append('file', file)
    return http.post<unknown, Attachment>('/files/upload', fd, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: (e) => onProgress?.(e.total ? Math.round((e.loaded / e.total) * 100) : 0),
    })
  },
  downloadUrl: (id: number) => `/api/files/${id}/download`,
  delete: (id: number) => http.delete(`/files/${id}`),
}

// ============================ Notifications ============================
export const notificationsApi = {
  unread: () => http.get<unknown, NotificationDigest>('/notifications/unread'),
  markRead: (messageId: number) => http.post(`/notifications/messages/${messageId}/read`),
}
