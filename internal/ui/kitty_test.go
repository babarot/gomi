package ui

import (
	"image"
	"testing"
)

func TestImageCells(t *testing.T) {
	cell := image.Pt(10, 20)
	tests := []struct {
		name       string
		pw, ph     int
		w, h       int
		cell       image.Point
		cols, rows int
	}{
		{name: "wide image fits the width", pw: 1100, ph: 700, w: 37, h: 29, cell: cell, cols: 37, rows: 12},
		{name: "tall image fits the height", pw: 400, ph: 2000, w: 60, h: 29, cell: cell, cols: 12, rows: 29},
		{name: "small image is not stretched", pw: 40, ph: 40, w: 60, h: 29, cell: cell, cols: 4, rows: 2},
		{name: "square cells", pw: 1000, ph: 1000, w: 60, h: 29, cell: image.Pt(20, 20), cols: 29, rows: 29},
		{name: "tiny image takes a cell", pw: 1, ph: 1, w: 60, h: 29, cell: cell, cols: 1, rows: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols, rows := imageCells(tt.pw, tt.ph, tt.w, tt.h, tt.cell)
			if cols != tt.cols || rows != tt.rows {
				t.Errorf("imageCells = %dx%d, want %dx%d", cols, rows, tt.cols, tt.rows)
			}
		})
	}
}
