package query

// QueryRequest representa o corpo da requisição SQL Explorer.
type QueryRequest struct {
	SQL string `json:"sql"`
}

// QueryResponse representa o resultado de uma query executada.
type QueryResponse struct {
	Columns   []string `json:"columns"`
	Rows      [][]any  `json:"rows"`
	RowCount  int      `json:"row_count"`
	Truncated bool     `json:"truncated"`
}
