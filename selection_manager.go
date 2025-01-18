package main

var (
	selectionManager = &SelectionManager{items: []File{}}
)

type SelectionManager struct {
	items []File
}

func (r *SelectionManager) Add(item File) {
	if r.Contains(item) {
		return
	}
	r.items = append(r.items, item)
}

func (r *SelectionManager) Remove(item File) {
	index := r.IndexOf(item)
	if index == -1 {
		return
	}
	r.items = append(r.items[:index], r.items[index+1:]...)
}

func (r *SelectionManager) Contains(item File) bool {
	return r.IndexOf(item) != -1
}

func (r *SelectionManager) IndexOf(item File) int {
	for i, v := range r.items {
		if v == item {
			return i
		}
	}
	return -1
}
