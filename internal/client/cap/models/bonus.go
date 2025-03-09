package models

type LabelTitle struct {
	Language string `json:"language"`
	Value    string `json:"value"`
}

type CreateLabelRequestBody struct {
	Color       string       `json:"color"`
	Titles      []LabelTitle `json:"titles"`
	Description string       `json:"description"`
}

type CreateLabelResponseBody struct {
	UUID string `json:"uuid"`
}

type GetLabelResponseBody struct {
	UUID           string       `json:"uuid"`
	Color          string       `json:"color"`
	Node           string       `json:"node"`
	UserID         string       `json:"userId"`
	AuthorCreation string       `json:"authorCreation"`
	AuthorEditing  string       `json:"authorEditing"`
	CreatedAt      string       `json:"createdAt"`
	UpdatedAt      string       `json:"updatedAt"`
	Titles         []LabelTitle `json:"titles"`
	Description    string       `json:"description"`
}
