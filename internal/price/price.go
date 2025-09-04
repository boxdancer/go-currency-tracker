package price

import "context"

// PriceClient описывает минимальный контракт клиента цен.
// Интерфейс находится в отдельном пакете, чтобы избежать циклических импортов.
type PriceClient interface {
	GetPrice(ctx context.Context, id, vs string) (float64, error)
}
