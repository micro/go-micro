package proto

type Message struct {
	Data []byte
}

func (m *Message) MarshalJSON() ([]byte, error) {
	return m.Data, nil
}

func (m *Message) UnmarshalJSON(data []byte) error {
	m.Data = data
	return nil
}

func (m *Message) ProtoMessage() {}

func (m *Message) Reset() {
	*m = Message{}
}

func (m *Message) String() string {
	return string(m.Data)
}

func (m *Message) Marshal() ([]byte, error) {
	return m.Data, nil
}

func (m *Message) Unmarshal(data []byte) error {
	m.Data = data
	return nil
}

func NewMessage(data []byte) *Message {
	return &Message{data}
}
