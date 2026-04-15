package models

type QRPayload struct {
	T  string `json:"t"`
	KB string `json:"kb"`
}

type ScanQRRequest struct {
	QR string `json:"qr" validate:"required"`
}
