import api from './client'

export interface QueryResult {
  columns:   string[]
  rows:      unknown[][]
  row_count: number
  truncated: boolean
}

export const queryApi = {
  execute: (grupoId: string, sql: string) =>
    api.post<{ success: boolean; message: string; data: QueryResult }>(
      `/admin/grupos/${grupoId}/query`,
      { sql }
    )
}
