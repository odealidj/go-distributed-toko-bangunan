package model

type Product struct {
	ID            string
	CategoryID    string
	CategoryName  string
	SKU           string
	Name          string
	Brand         string
	Unit          string
	Price         int64
	WeightKG      float64
	RequiresTruck bool
	AvailableQty  float64
}

type ProductFilter struct {
	CategoryID string
	Search     string
	Page       int
	PerPage    int
}

type ValidatedItem struct {
	ProductID   string
	ProductName string
	Unit        string
	Quantity    float64
	UnitPrice   int64
	LineTotal   int64
}

type OrderItemInput struct {
	ProductID string
	Quantity  float64
}
