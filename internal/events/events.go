package events

type Message struct {
	Topic   string
	Key     []byte
	Value   []byte
	Headers map[string][]byte
}
