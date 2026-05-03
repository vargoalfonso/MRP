package models

import "time"

type ImportData struct {
	ID        uint      `json:"-" gorm:"primaryKey"`
	Uniq      string    `json:"uniq" gorm:"type:varchar(50)"`
	Period    string    `json:"period" gorm:"type:char(7)"`
	Value     float64   `json:"value" gorm:"type:decimal(20,2)"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}
