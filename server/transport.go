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

func newTransports(cfg *config.Backend) []Transport {
	// Make sure serial not affected by connections parameter
	conns := cfg.Connections
	if cfg.Protocol == "serial" {
		conns = 1
	}
	ret := make([]Transport, conns)
	for i := 0; i < conns; i++ {
		ret[i] = newTransport(cfg)
	}
	return ret
}

func modbusErrorPdu(req *pdu, errCode uint8) *pdu {
	return &pdu{
		unitID:   req.unitID,
		funcCode: (0x80 | req.funcCode),
		payload:  []byte{errCode},
	}
}
