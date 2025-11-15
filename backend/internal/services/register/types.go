package register

type Data struct {
	Address   string `json:"address"`
	City      string `json:"city"`
	Country   string `json:"country"`
	Course    string `json:"course"`
	Discover  string `json:"discover"`
	DNI       string `json:"dni"`
	DOB       string `json:"dob"`
	Email     string `json:"email"`
	IBAN      string `json:"iban"`
	Locality  string `json:"locality"`
	Modality  string `json:"modality"`
	Name      string `json:"name"`
	Payment   string `json:"payment"`
	Phone     string `json:"phone"`
	Postal    string `json:"postalcode"`
	Signature string `json:"signature"`
	StartDate string `json:"startdate"`
	Surname   string `json:"surname"`
	Website   string `json:"website,omitempty"`
}

type Result struct {
	Message string `json:"message"`
}
