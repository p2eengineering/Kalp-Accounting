package models

type Response struct {
	Status     string      `json:"status"`
	StatusCode uint        `json:"statusCode"`
	Success    bool        `json:"success"`
	Message    string      `json:"message"`
	Response   interface{} `json:"response"`
}
type UserRole struct {
	Id      string `json:"user"`
	Role    string `json:"role"`
	DocType string `json:"docType"`
	Desc    string `json:"desc"`
}

type Sender struct {
	Sender string `json:"sender"`
}

type Utxo struct {
	Key     string `json:"_id,omitempty"`
	Account string `json:"account"`
	DocType string `json:"docType"`
	Amount  string `json:"amount"`
}

type Allow struct {
	Owner   string `json:"id"`
	Amount  string `json:"amount"`
	DocType string `json:"docType"`
	Spender string `json:"account"`
}

type TransferSingle struct {
	Operator string      `json:"address"`
	From     string      `json:"from"`
	To       string      `json:"to"`
	ID       string      `json:"id"`
	Value    interface{} `json:"value"`
}

type Account struct {
	Recipient string `json:"recipient"`
}
