package models

type CreateKanbanParameterRequest struct {
	ItemUniqCode string `json:"item_uniq_code" validate:"required"`
	KanbanQty    int    `json:"kanban_qty" validate:"required"`
	MinStock     int    `json:"min_stock"`
	MaxStock     int    `json:"max_stock"`
	Status       string `json:"status"`
}

type UpdateKanbanParameterRequest struct {
	KanbanQty int     `json:"kanban_qty"`
	MinStock  int     `json:"min_stock"`
	MaxStock  int     `json:"max_stock"`
	Status    *string `json:"status"`
}
