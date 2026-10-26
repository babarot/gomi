package trash

import "sync/atomic"

// Progress counts the items of a listing: how many there are, as far as
// the listing has found, and how many of them are read. Storages count
// into it while they list, from several goroutines; a UI reads it to show
// how far the listing has got. A nil Progress counts nothing.
type Progress struct {
	total atomic.Int64
	read  atomic.Int64
}

// AddTotal adds n items found to list
func (p *Progress) AddTotal(n int) {
	if p != nil {
		p.total.Add(int64(n))
	}
}

// AddRead adds n items read
func (p *Progress) AddRead(n int) {
	if p != nil {
		p.read.Add(int64(n))
	}
}

// Counts returns how many items are read and how many there are so far
func (p *Progress) Counts() (read, total int) {
	if p == nil {
		return 0, 0
	}
	return int(p.read.Load()), int(p.total.Load())
}

// progressLister is a Storage that counts its listing into a Progress
type progressLister interface {
	ListWithProgress(p *Progress) ([]*File, error)
}
