package store

// Todos os valores monetários trafegam como centavos (int64). Isso evita os
// erros de arredondamento que float64 introduz ao somar preços.

type Client struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Address    string `json:"address"`
	Notes      string `json:"notes"`
	OrderCount int    `json:"orderCount"`
}

type Service struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Category   string `json:"category"`
	PriceCents int64  `json:"priceCents"`
	Time       string `json:"time"`
}

type OrderItem struct {
	ID         int64  `json:"id"`
	ServiceID  *int64 `json:"serviceId"`
	Name       string `json:"name"`
	Qty        int64  `json:"qty"`
	PriceCents int64  `json:"priceCents"`
}

type Order struct {
	ID         int64       `json:"id"`
	Number     string      `json:"number"`
	ClientID   int64       `json:"clientId"`
	ClientName string      `json:"clientName"`
	Date       string      `json:"date"`
	Due        string      `json:"due"`
	Status     string      `json:"status"`
	Payment    string      `json:"payment"`
	Paid       bool        `json:"paid"`
	Notes      string      `json:"notes"`
	Items      []OrderItem `json:"items"`
	TotalCents int64       `json:"totalCents"`
}

type Transaction struct {
	ID          int64  `json:"id"`
	Date        string `json:"date"`
	Type        string `json:"type"`
	Description string `json:"description"`
	ValueCents  int64  `json:"valueCents"`
	Payment     string `json:"payment"`
}

// Summary alimenta o Dashboard e a página Financeiro numa única chamada.
type Summary struct {
	RevenueCents    int64   `json:"revenueCents"`
	ReceivableCents int64   `json:"receivableCents"`
	ExpenseCents    int64   `json:"expenseCents"`
	BalanceCents    int64   `json:"balanceCents"`
	InProgress      int     `json:"inProgress"`
	Ready           int     `json:"ready"`
	RecentOrders    []Order `json:"recentOrders"`
	UpcomingOrders  []Order `json:"upcomingOrders"`
}

// Backup é o formato de exportação/importação em JSON.
type Backup struct {
	Version      int           `json:"version"`
	ExportedAt   string        `json:"exportedAt"`
	Clients      []Client      `json:"clients"`
	Services     []Service     `json:"services"`
	Orders       []Order       `json:"orders"`
	Transactions []Transaction `json:"transactions"`
}

// OrderStatuses são os únicos status aceitos, na ordem em que aparecem na UI.
var OrderStatuses = []string{
	"Orçamento", "Aguardando", "Em produção", "Pronto", "Entregue", "Cancelado",
}

func ValidStatus(s string) bool {
	for _, v := range OrderStatuses {
		if v == s {
			return true
		}
	}
	return false
}
