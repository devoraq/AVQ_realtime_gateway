// Package events описывает транспортные сообщения, которыми обмениваются слои приложения.
package events

// Message представляет универсальный транспортный формат сообщения для событийной шины.
type Message struct {
	Topic   string
	Key     []byte
	Value   []byte
	Headers map[string][]byte
}
