-- +migrate Up
-- Fix: "column total_po_incoming of relation delivery_notes does not exist" saat membuat DN.
--
-- Model delivery_note.DeliveryNote memakai kolom total_po_incoming, tetapi migrasi 0015
-- hanya menambah total_po_qty, total_dn_created, dan total_dn_incoming.
--
-- Migrasi ini juga memastikan dua kolom delivery_note_items yang dipakai model
-- (qty_sent dan "check") ada. Semua memakai IF NOT EXISTS, jadi aman bila kolomnya
-- sudah ada di database Anda.
ALTER TABLE delivery_notes
  ADD COLUMN IF NOT EXISTS total_po_incoming INT NOT NULL DEFAULT 0;

ALTER TABLE delivery_note_items
  ADD COLUMN IF NOT EXISTS qty_sent BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS "check" VARCHAR(64) NOT NULL DEFAULT '';
