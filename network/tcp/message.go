package tcp

var (
	defaultMsgMinSize = 1
)

type (
	MessageParserConf struct {
		// Type of the message length field (0: uint16, 1: uint32, 2: uint64)
		MsgLengthType LenType
		// Minimum message size
		MsgMinSize int
		// Maximum message size (0: no limit)
		MsgMaxSize   int
		LittleEndian bool
	}
	MessageParser struct {
		conf *MessageParserConf
	}
)
