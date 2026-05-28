package response

import "testing"

func TestNewPagination(t *testing.T) {
	pagination := NewPagination(2, 10, 25)

	if pagination.Page != 2 {
		t.Fatalf("expected page 2, got %d", pagination.Page)
	}
	if pagination.PerPage != 10 {
		t.Fatalf("expected per page 10, got %d", pagination.PerPage)
	}
	if pagination.TotalPages != 3 {
		t.Fatalf("expected total pages 3, got %d", pagination.TotalPages)
	}
	if !pagination.HasNext {
		t.Fatal("expected has next")
	}
	if !pagination.HasPrev {
		t.Fatal("expected has prev")
	}
}

func TestNewPaginationDefaults(t *testing.T) {
	pagination := NewPagination(0, 0, -1)

	if pagination.Page != 1 {
		t.Fatalf("expected default page 1, got %d", pagination.Page)
	}
	if pagination.PerPage != 10 {
		t.Fatalf("expected default per page 10, got %d", pagination.PerPage)
	}
	if pagination.TotalItems != 0 {
		t.Fatalf("expected total items 0, got %d", pagination.TotalItems)
	}
	if pagination.TotalPages != 0 {
		t.Fatalf("expected total pages 0, got %d", pagination.TotalPages)
	}
}
