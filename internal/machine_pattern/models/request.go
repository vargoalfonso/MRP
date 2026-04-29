package models

type CreateMachinePatternRequest struct {
	UniqCode     string   `json:"uniq_code" validate:"required,max=64"`
	MachineID   int64    `json:"machine_id" validate:"required,gt=0"`
	CycleTime    *float64 `json:"cycle_time" validate:"omitempty,gt=0"`
	PatternValue *float64 `json:"pattern_value" validate:"omitempty,gt=0"`
	WorkingDays  *int     `json:"working_days" validate:"omitempty,gt=0"`
	MovingType   string   `json:"moving_type" validate:"omitempty,oneof='Fast Moving' 'Slow Moving' 'Normal'"`
	MinOutput    *float64 `json:"min_output" validate:"omitempty,gte=0"`
	PRLReference *float64 `json:"prl_reference" validate:"omitempty,gte=0"`
	Status       string   `json:"status" validate:"omitempty,oneof=Active Inactive"`
}

type UpdateMachinePatternRequest struct {
	CycleTime    *float64 `json:"cycle_time" validate:"omitempty,gt=0"`
	PatternValue *float64 `json:"pattern_value" validate:"omitempty,gt=0"`
	WorkingDays  *int     `json:"working_days" validate:"omitempty,gt=0"`
	MovingType   string   `json:"moving_type" validate:"omitempty,oneof='Fast Moving' 'Slow Moving' 'Normal'"`
	MinOutput    *float64 `json:"min_output" validate:"omitempty,gte=0"`
	PRLReference *float64 `json:"prl_reference" validate:"omitempty,gte=0"`
	Status       string   `json:"status" validate:"omitempty,oneof=Active Inactive"`
}

type BulkMachinePatternItem struct {
	UniqCode     string   `json:"uniq_code" validate:"required,max=64"`
	MachineID    int64    `json:"machine_id" validate:"required,gt=0"`
	CycleTime    *float64 `json:"cycle_time" validate:"omitempty,gt=0"`
	PatternValue *float64 `json:"pattern_value" validate:"omitempty,gt=0"`
	WorkingDays  *int     `json:"working_days" validate:"omitempty,gt=0"`
	MovingType   string   `json:"moving_type" validate:"omitempty,oneof='Fast Moving' 'Slow Moving' 'Normal'"`
	MinOutput    *float64 `json:"min_output" validate:"omitempty,gte=0"`
	PRLReference *float64 `json:"prl_reference" validate:"omitempty,gte=0"`
}

type BulkMachinePatternRequest struct {
	Items []BulkMachinePatternItem `json:"items" validate:"required,min=1,dive"`
}

type BulkMachinePatternResponse struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
}
