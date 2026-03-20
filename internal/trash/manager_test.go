package trash

import (
	"testing"
)

func TestDetermineStrategy(t *testing.T) {
	tests := []struct {
		name     string
		storages []Storage
		want     Strategy
	}{
		{
			name:     "no storages",
			storages: nil,
			want:     StrategyNone,
		},
		{
			name: "single xdg storage",
			storages: []Storage{
				&mockStorage{storageType: StorageTypeXDG},
			},
			want: StrategyXDG,
		},
		{
			name: "single legacy storage",
			storages: []Storage{
				&mockStorage{storageType: StorageTypeLegacy},
			},
			want: StrategyLegacy,
		},
		{
			name: "multiple storages",
			storages: []Storage{
				&mockStorage{storageType: StorageTypeXDG},
				&mockStorage{storageType: StorageTypeLegacy},
			},
			want: StrategyAuto,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := determineStrategy(tt.storages); got != tt.want {
				t.Errorf("determineStrategy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_ListStorages(t *testing.T) {
	m := &Manager{
		storages: []Storage{
			&mockStorage{storageType: StorageTypeXDG, available: true},
			&mockStorage{storageType: StorageTypeLegacy, available: false},
		},
	}

	infos := m.ListStorages()
	if len(infos) != 2 {
		t.Fatalf("expected 2 storage infos, got %d", len(infos))
	}
	if infos[0].Type != StorageTypeXDG {
		t.Errorf("first storage type = %v, want XDG", infos[0].Type)
	}
	if infos[1].Type != StorageTypeLegacy {
		t.Errorf("second storage type = %v, want Legacy", infos[1].Type)
	}
}

func TestManager_IsPrimaryStorageAvailable(t *testing.T) {
	tests := []struct {
		name     string
		storages []Storage
		want     bool
	}{
		{
			name:     "no storages",
			storages: nil,
			want:     false,
		},
		{
			name: "primary available",
			storages: []Storage{
				&mockStorage{available: true},
			},
			want: true,
		},
		{
			name: "primary unavailable",
			storages: []Storage{
				&mockStorage{available: false},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{storages: tt.storages}
			if got := m.IsPrimaryStorageAvailable(); got != tt.want {
				t.Errorf("IsPrimaryStorageAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

// mockStorage implements Storage for testing
type mockStorage struct {
	storageType StorageType
	available   bool
	trashes     []string
	files       []*File
	putErr      error
	restoreErr  error
	removeErr   error
	listErr     error
}

func (m *mockStorage) Put(src string) error        { return m.putErr }
func (m *mockStorage) Restore(f *File, dst string) error { return m.restoreErr }
func (m *mockStorage) Remove(f *File) error         { return m.removeErr }

func (m *mockStorage) List() ([]*File, error) {
	return m.files, m.listErr
}

func (m *mockStorage) Info() *StorageInfo {
	return &StorageInfo{
		Type:      m.storageType,
		Available: m.available,
		Trashes:   m.trashes,
	}
}
