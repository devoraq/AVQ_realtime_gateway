package handlers

import "fmt"

// HandlerError описывает типовую ошибку обработчиков входящих сообщений
// с кодом, пользовательским сообщением и дополнительными деталями.
type HandlerError struct {
	Code    string
	Message string
	Details string
}

func (e HandlerError) Error() string {
	if e.Details == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
}

// WithDetails возвращает копию ошибки, дополненную деталями.
func (e HandlerError) WithDetails(err error) HandlerError {
	if err == nil {
		return e
	}
	e.Details = err.Error()
	return e
}

var (
	// ErrMessageInvalidPayload сигнализирует о неверной структуре входящего сообщения.
	ErrMessageInvalidPayload = HandlerError{
		Code:    "message_invalid_payload",
		Message: "payload is invalid",
	}
	// ErrMessageValidationFailed используется при ошибке бизнес-валидации сообщения.
	ErrMessageValidationFailed = HandlerError{
		Code:    "message_validation_failed",
		Message: "message validation failed",
	}
	// ErrMessageDeliveryFailed отражает сбой доставки сообщения получателю.
	ErrMessageDeliveryFailed = HandlerError{
		Code:    "message_delivery_failed",
		Message: "failed to deliver message",
	}
)
