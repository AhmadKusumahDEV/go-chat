package response

type ApiResponse struct {
	Status  int16 `json:"status"`
	Data    any   `json:"data"`
	Message any   `json:"message"`
}
