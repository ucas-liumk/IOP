// Envelope must match server/internal/interface/apiresp/response.go.
export interface Envelope<T = unknown> {
  code: number;
  data?: T;
  error?: {
    code: string;
    message: string;
    kind: string;
  };
  trace_id: string;
}
