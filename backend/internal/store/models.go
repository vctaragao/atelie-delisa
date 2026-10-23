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

// OrderItem é um serviço dentro do pedido. Piece é a peça de roupa em que ele
// será feito — "calça jeans", "vestido de festa" —, texto livre e opcional.
type OrderItem struct {
	ID         int64  `json:"id"`
	ServiceID  *int64 `json:"serviceId"`
	Name       string `json:"name"`
	Piece      string `json:"piece"`
	Qty        int64  `json:"qty"`
	PriceCents int64  `json:"priceCents"`
}

// Order traz o pedido com a conta já fechada. Os percentuais são o que a
// Maria combina com a cliente; os valores em centavos são derivados deles
// pela view order_totals, e por isso só saem do banco — mandá-los numa
// alteração não muda nada.
type Order struct {
	ID              int64       `json:"id"`
	Number          string      `json:"number"`
	ClientID        int64       `json:"clientId"`
	ClientName      string      `json:"clientName"`
	Date            string      `json:"date"`
	Due             string      `json:"due"`
	DueTime         string      `json:"dueTime"`
	Status          string      `json:"status"`
	Payment         string      `json:"payment"`
	Paid            bool        `json:"paid"`
	Notes           string      `json:"notes"`
	DiscountPercent int64       `json:"discountPercent"`
	RushPercent     int64       `json:"rushPercent"`
	Items           []OrderItem `json:"items"`
	ItemsCents      int64       `json:"itemsCents"`
	DiscountCents   int64       `json:"discountCents"`
	RushCents       int64       `json:"rushCents"`
	TotalCents      int64       `json:"totalCents"`
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

// ---------- relatório do período ----------

// ReportService é uma linha do "serviços mais pedidos". O valor é o dos itens
// (quantidade × preço), antes do desconto e da urgência, que são do pedido
// inteiro e não de um serviço específico.
type ReportService struct {
	Name       string `json:"name"`
	Qty        int64  `json:"qty"`
	Orders     int    `json:"orders"`
	TotalCents int64  `json:"totalCents"`
}

// ReportClient é uma linha do "clientes que mais gastaram", já com desconto e
// urgência aplicados — é o que a cliente pagou ou vai pagar.
type ReportClient struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Orders     int    `json:"orders"`
	TotalCents int64  `json:"totalCents"`
}

type ReportStatus struct {
	Status string `json:"status"`
	Orders int    `json:"orders"`
}

// Report é o fechamento de um período, da data inicial à final, inclusive.
//
// Um pedido entra no período pela sua data de entrada. Pedidos cancelados
// ficam fora de todos os valores — só aparecem na contagem por status.
type Report struct {
	From string `json:"from"`
	To   string `json:"to"`

	Orders        int   `json:"orders"`
	Pieces        int64 `json:"pieces"`
	ClientsServed int   `json:"clientsServed"`

	OrdersCents     int64 `json:"ordersCents"`     // valor dos pedidos do período
	AverageCents    int64 `json:"averageCents"`    // valor médio por pedido
	PaidCents       int64 `json:"paidCents"`       // o que já foi pago
	ReceivableCents int64 `json:"receivableCents"` // o que falta receber
	DiscountCents   int64 `json:"discountCents"`   // desconto concedido
	RushCents       int64 `json:"rushCents"`       // acréscimo de urgência

	IncomeCents  int64 `json:"incomeCents"`  // entradas lançadas no financeiro
	ExpenseCents int64 `json:"expenseCents"` // saídas lançadas no financeiro
	ResultCents  int64 `json:"resultCents"`  // pago + entradas − saídas

	TopServices []ReportService `json:"topServices"`
	TopClients  []ReportClient  `json:"topClients"`
	ByStatus    []ReportStatus  `json:"byStatus"`
}
