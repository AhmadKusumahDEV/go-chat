package response

type ApiResponse struct {
	Status  int16 `json:"success"`
	Data    any   `json:"data"`
	Message any   `json:"error"`
}
