package ws

// MessageTypeError задаёт тип конверта для сообщений об ошибках.
const MessageTypeError = "error"

// Набор кодов ошибок, которые использует транспорт WebSocket.
const (
	ErrorCodeInternal        = "internal_error"
	ErrorCodeInvalidEnvelope = "invalid_envelope"
	ErrorCodeRouteNotFound   = "route_not_found"
)

// ErrorPayload описывает JSON-полезную нагрузку, которую получает клиент в случае ошибки транспорта.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
