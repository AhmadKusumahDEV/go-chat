package request

type PlanRequest struct {
	Plan string `json:"plan" binding:"required,oneof=premium standard basic"`
}
