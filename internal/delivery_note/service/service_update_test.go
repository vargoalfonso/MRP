package service

import (
	"testing"
	"time"

	"github.com/ganasa18/go-template/internal/delivery_note/models"
)

func TestDnItemProcessed(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name string
		it   models.DeliveryNoteItem
		want bool
	}{
		{"baru dibuat (check=progress)", models.DeliveryNoteItem{Check: "progress"}, false},
		{"check kosong (data lama)", models.DeliveryNoteItem{Check: ""}, false},
		{"huruf besar/spasi", models.DeliveryNoteItem{Check: " Progress "}, false},
		{"sudah incoming", models.DeliveryNoteItem{Check: "incoming"}, true},
		{"remaining", models.DeliveryNoteItem{Check: "remaining"}, true},
		{"completed", models.DeliveryNoteItem{Check: "completed"}, true},
		{"outgoing subcon", models.DeliveryNoteItem{Check: "outgoing"}, true},
		{"qty diterima > 0", models.DeliveryNoteItem{Check: "progress", QtyReceived: 5}, true},
		{"qty terkirim > 0", models.DeliveryNoteItem{Check: "progress", QtySent: 1}, true},
		{"received_at terisi", models.DeliveryNoteItem{Check: "progress", ReceivedAt: &now}, true},
	}
	for _, c := range cases {
		if got := dnItemProcessed(c.it); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}
