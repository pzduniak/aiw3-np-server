package structs

type PacketData struct {
	Header  PacketHeader
	Content []byte
}

type PacketHeader struct {
	Signature uint32
	Length    uint32
	Type      uint32
	Id        uint32
}
