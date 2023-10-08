package server

import (
	"errors"

	"github.com/blacktear23/modbus_gateway/config"
)

var (
	ErrBadCRC        = errors.New("Bad CRC")
	ErrShortFrame    = errors.New("Short frame")
	ErrProtocolError = errors.New("Invalid protocol")
)

func newTransport(cfg *config.Backend) Transport {
	switch cfg.Protocol {
	case "tcp", "tls":
		return newTcpTransport(cfg)
	case "serial":
		return newSerialTransport(cfg)
	}
	return nil
}

func modbusErrorPdu(req *pdu, errCode uint8) *pdu {
	return &pdu{
		unitID:   req.unitID,
		funcCode: (0x80 | req.funcCode),
		payload:  []byte{errCode},
	}
}
