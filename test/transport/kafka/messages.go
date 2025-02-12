package kafka

type Brand struct {
	Message struct {
		EventType string `json:"eventType"`
	} `json:"message"`
	Brand struct {
		UUID           string            `json:"uuid"`
		LocalizedNames map[string]string `json:"localized_names"`
		Alias          string            `json:"alias"`
		ProjectID      string            `json:"project_id"`
		StatusEnabled  bool              `json:"status_enabled"`
		CreatedAt      int64             `json:"created_at"`
	} `json:"brand"`
}
