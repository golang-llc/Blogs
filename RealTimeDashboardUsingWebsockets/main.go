package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

const (
	ORDERS    = "orders"
	CUSTOMERS = "customers"
	PRODUCTS  = "products"
)

type DashboardData struct {
	Orders    int `json:"orders"`
	Customers int `json:"customers"`
	Products  int `json:"products"`
}

func (Dashboard *DashboardData) FetchDashboardHelper() []byte {
	data, err := json.Marshal(Dashboard)
	if err != nil {
		log.Printf("Error: %v", err)
	}
	return data
}

func (Dashboard *DashboardData) AddDashboardData(s string) {
	switch s {
	case ORDERS:
		Dashboard.Orders++
	case CUSTOMERS:
		Dashboard.Customers++
	case PRODUCTS:
		Dashboard.Products++
	}
	// update presently connected websocket connections about the dashboard changes
	go func() {
		update <- 1
	}()
}

func (Dashboard *DashboardData) RemoveDashboardData(s string) error {
	switch s {
	case ORDERS:
		if Dashboard.Orders == 0 {
			return fmt.Errorf("no order remains to remove")
		}
		Dashboard.Orders--
	case CUSTOMERS:
		if Dashboard.Customers == 0 {
			return fmt.Errorf("no customer remains to remove")
		}
		Dashboard.Customers--
	case PRODUCTS:
		if Dashboard.Products == 0 {
			return fmt.Errorf("no product remains to remove")
		}
		Dashboard.Products--
	}
	// update presently connected websocket connections about the dashboard changes
	go func() {
		update <- 1
	}()
	return nil
}

var update chan int
var dashboard DashboardData

func main() {
	fmt.Println("Starting Server...")
	update = make(chan int)
	router := chi.NewRouter()
	router.Route("/", func(ws chi.Router) {
		ws.Get("/", Alive)
		ws.Get("/dashboard", DashboardHandler)
		ws.Post("/sign-up", AddCustomer)
		ws.Post("/order", AddOrder)
		ws.Post("/product", AddProducts)
		ws.Delete("/order", RemoveOrder)
		ws.Delete("/product", RemoveProducts)
		ws.Delete("/sign-off", RemoveCustomer)

	})
	log.Fatal(http.ListenAndServe(":8082", router))
}

func Alive(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("<h4>Welcome to Dashboard app</h4>"))
	if err != nil {
		return
	}
	log.Printf("Welcome to chatting app")
	return
}

var upgrades = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	upgrades.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrades.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	var data []byte
	data = dashboard.FetchDashboardHelper()
	conn.WriteMessage(1, data)
	for {
		select {
		case <-update:
			data = dashboard.FetchDashboardHelper()
			conn.WriteMessage(1, data)
		}
	}
	return
}

func AddOrder(w http.ResponseWriter, r *http.Request) {
	dashboard.AddDashboardData(ORDERS)
	_, err := w.Write([]byte("Order Added"))
	if err != nil {
		return
	}
	return
}
func AddCustomer(w http.ResponseWriter, r *http.Request) {
	dashboard.AddDashboardData(CUSTOMERS)
	_, err := w.Write([]byte("Customer Added"))
	if err != nil {
		return
	}
	return
}
func AddProducts(w http.ResponseWriter, r *http.Request) {
	dashboard.AddDashboardData(PRODUCTS)
	_, err := w.Write([]byte("Product Added"))
	if err != nil {
		return
	}
	return
}

func RemoveOrder(w http.ResponseWriter, r *http.Request) {
	if dashboard.Orders == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err := dashboard.RemoveDashboardData(ORDERS)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, err = w.Write([]byte("Order Canceled"))
	if err != nil {
		return
	}
	return
}
func RemoveCustomer(w http.ResponseWriter, r *http.Request) {
	if dashboard.Customers == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err := dashboard.RemoveDashboardData(CUSTOMERS)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, err = w.Write([]byte("Customer Removed"))
	if err != nil {
		return
	}
	return
}
func RemoveProducts(w http.ResponseWriter, r *http.Request) {
	if dashboard.Customers == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err := dashboard.RemoveDashboardData(PRODUCTS)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, err = w.Write([]byte("Product Removed"))
	if err != nil {
		return
	}
	return
}
