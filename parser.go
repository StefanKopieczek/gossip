package gossip

type MessageParser interface {
    ParseMessage(rawData []byte) (SipMessage, error)
}
func NewMessageParser() (MessageParser) {
    return nil // TODO
}
