package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// product structure
type product struct {
	ID          int     `json:"id"`
	Price       float64 `json:"price"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
}

// pagination structure
type Pagination struct {
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages"`
	TotalItems  int `json:"total_items"`
	Limit       int `json:"limit"`
	Offset      int `json:"offset"`
}

// pagination response structure
type PaginatedResponse struct {
	Data       []product  `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type PriceResponse struct {
	Name       string  `json:"name"`
	Quantity   int     `json:"quantity"`
	TotalPrice float64 `json:"total_price"`
}

// array with dummy data
var products = []product{
	{ID: 1, Price: 50, Name: "kaas", Description: "een lekker stuk kaas", Category: "zuivel"},
	{ID: 2, Price: 10, Name: "t-shirt", Description: "een simpel wit t-shirt", Category: "shirts"},
	{ID: 3, Price: 35, Name: "nike air max", Description: "mooie stijlvolle schoenen", Category: "schoenen"},
}

// function to filter the array
func filterProducts(products []product, maxPrice float64, minPrice float64, name string, sortBy string, category string) []product {
	var filtered []product
	for _, p := range products {
		if (minPrice <= 0 || p.Price >= minPrice) &&
			(maxPrice <= 0 || p.Price <= maxPrice) &&
			(name == "" || strings.Contains(strings.ToLower(p.Name), strings.ToLower(name))) &&
			(category == "" || strings.EqualFold(p.Category, category)) {
			filtered = append(filtered, p)
		}
	}

	if sortBy == "LnH" {
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Price < filtered[j].Price
		})
	} else if sortBy == "HnL" {
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Price > filtered[j].Price
		})
	}

	return filtered
}

// function to get all products
func getProducts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	limit, err := strconv.Atoi(r.FormValue("limit"))
	if err != nil || limit < 1 {
		http.Error(w, "400 invalid limit value", http.StatusBadRequest)
		return
	}
	offset, err := strconv.Atoi(r.FormValue("offset"))
	if err != nil || offset < 0 {
		http.Error(w, "400 invalid offset value", http.StatusBadRequest)
		return
	}

	minPriceStr := r.FormValue("min_price")
	maxPriceStr := r.FormValue("max_price")
	name := r.FormValue("name")
	sortBy := r.FormValue("sort_by")
	category := r.FormValue("category")

	minPrice := 0.0
	if minPriceStr != "" {
		minPrice, err = strconv.ParseFloat(minPriceStr, 64)
		if err != nil || minPrice < 0 {
			http.Error(w, "400 invalid minimum price value", http.StatusBadRequest)
			return
		}
	}
	maxPrice := 0.0
	if maxPriceStr != "" {
		maxPrice, err = strconv.ParseFloat(maxPriceStr, 64)
		if err != nil || maxPrice < 0 {
			http.Error(w, "400 invalid maximum price value", http.StatusBadRequest)
			return
		}
	}

	if maxPrice < minPrice && minPrice > 0 {
		http.Error(w, "400 maximum price cannot be less than minimum price", http.StatusBadRequest)
		return
	}
	if sortBy != "" && sortBy != "HnL" && sortBy != "LnH" {
		http.Error(w, "400 invalid sort_by value", http.StatusBadRequest)
		return
	}

	if category != "" {
		categoryExists := false
		for _, p := range products {
			if strings.EqualFold(p.Category, category) {
				categoryExists = true
				break
			}
		}
		if !categoryExists {
			http.Error(w, "400 invalid category value", http.StatusBadRequest)
			return
		}
	}

	filteredProducts := filterProducts(products, maxPrice, minPrice, name, sortBy, category)

	end := offset + limit
	if end > len(filteredProducts) {
		end = len(filteredProducts)
	}

	paginatedProducts := filteredProducts[offset:end]
	response := PaginatedResponse{
		Data: paginatedProducts,
		Pagination: Pagination{
			CurrentPage: (offset / limit) + 1,
			TotalPages:  (len(filteredProducts) + limit - 1) / limit,
			TotalItems:  len(filteredProducts),
			Limit:       limit,
			Offset:      offset,
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
	}
}

// function to get a single product
func getProduct(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "400 product id not found", http.StatusBadRequest)
		return
	}
	for _, item := range products {
		if item.ID == id {
			json.NewEncoder(w).Encode(item)
			return
		}
	}
	http.Error(w, "product not found", http.StatusNotFound)
}

func calculatePrice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var selectProduct struct {
		Name     string `json:"name"`
		Quantity int    `json:"quantity"`
	}
	err := json.NewDecoder(r.Body).Decode(&selectProduct)
	if err != nil {
		http.Error(w, "400 id not found", http.StatusBadRequest)
		return
	}
	var foundProduct *product
	for _, p := range products {
		if p.Name == selectProduct.Name {
			foundProduct = &p
			break
		}
	}
	if foundProduct == nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}
	totalPrice := foundProduct.Price * float64(selectProduct.Quantity)

	response := PriceResponse{
		Name:       foundProduct.Name,
		Quantity:   selectProduct.Quantity,
		TotalPrice: totalPrice,
	}

	json.NewEncoder(w).Encode(response)
}

// function to add product
func addProduct(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var newProduct product
	_ = json.NewDecoder(r.Body).Decode(&newProduct)
	maxId := 0
	for _, item := range products {
		if item.ID > maxId {
			maxId = item.ID
		}
	}
	newProduct.ID = maxId + 1
	products = append(products, newProduct)
	json.NewEncoder(w).Encode(newProduct)
}

// function to delete product
func deleteProduct(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "400 id not found", http.StatusBadRequest)
		return
	}
	for index, item := range products {
		if item.ID == id {
			products = append(products[:index], products[index+1:]...)
			break
		}
		http.Error(w, "product not found", http.StatusNotFound)
	}
}

// Function to update product
func updateProduct(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "400 id not found", http.StatusBadRequest)
		return
	}
	for index, item := range products {
		if item.ID == id {
			products = append(products[:index], products[index+1:]...)
			var newProduct product
			_ = json.NewDecoder(r.Body).Decode(&newProduct)
			newProduct.ID = id
			products = append(products, newProduct)
			json.NewEncoder(w).Encode(newProduct)
			return
		}
		http.Error(w, "product not found", http.StatusNotFound)
	}
}

func main() {
	// Router
	router := mux.NewRouter()

	// API endpoints
	router.HandleFunc("/products", getProducts).Methods("GET")
	router.HandleFunc("/products/{id}", getProduct).Methods("GET")
	router.HandleFunc("/products/{id}", updateProduct).Methods("PUT")
	router.HandleFunc("/products", addProduct).Methods("POST")
	router.HandleFunc("/products/{id}", deleteProduct).Methods("DELETE")
	router.HandleFunc("/products/calculatePrice", calculatePrice).Methods("POST")

	// port config
	log.Fatal(http.ListenAndServe(":9090", router))
}
