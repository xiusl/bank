package util

const (
	USD = "USD"
	EUR = "EUR"
)

func IsSupporedCurrency(currency string) bool {
	switch currency {
	case USD, EUR:
		return true
	}
	return false
}
