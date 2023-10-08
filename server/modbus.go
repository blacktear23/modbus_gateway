package server

import (
	"encoding/binary"
	"errors"
)

type pdu struct {
	unitID   uint8
	funcCode uint8
	payload  []byte
}

type mbap struct {
	txnID      uint16
	protocolID uint16
	unitID     uint8
}

type Endianness uint

var (
	ErrNeedReadMore = errors.New("Need read more byte")
)

const (
	// endianness of 16-bit registers
	BIG_ENDIAN    Endianness = 1
	LITTLE_ENDIAN Endianness = 2

	// Function Codes
	// coils
	FCReadCoils          uint8 = 0x01
	FCWriteSingleCoil    uint8 = 0x05
	FCWriteMultipleCoils uint8 = 0x0f

	// discrete inputs
	FCReadDiscreteInputs uint8 = 0x02

	// 16-bit input/holding registers
	FCReadHoldingRegisters       uint8 = 0x03
	FCReadInputRegisters         uint8 = 0x04
	FCWriteSingleRegister        uint8 = 0x06
	FCWriteMultipleRegisters     uint8 = 0x10
	FCMaskWriteRegister          uint8 = 0x16
	FCReadWriteMultipleRegisters uint8 = 0x17
	FCReadFifoQueue              uint8 = 0x18

	// file access
	FCReadFileRecord  uint8 = 0x14
	FCWriteFileRecord uint8 = 0x15

	// Error codes
	MErrIllegalFunction         uint8 = 0x01
	MErrIllegalDataAddress      uint8 = 0x02
	MErrIllegalDataValue        uint8 = 0x03
	MErrServerDeviceFailure     uint8 = 0x04
	MErrAcknowledge             uint8 = 0x05
	MErrServerDeviceBusy        uint8 = 0x06
	MErrMemoryParityError       uint8 = 0x08
	MErrGWPathUnavailable       uint8 = 0x0a
	MErrGWTargetFailedToRespond uint8 = 0x0b
)

func uint16ToBytes(endianness Endianness, in uint16) (out []byte) {
	out = make([]byte, 2)
	switch endianness {
	case BIG_ENDIAN:
		binary.BigEndian.PutUint16(out, in)
	case LITTLE_ENDIAN:
		binary.LittleEndian.PutUint16(out, in)
	}
	return
}

func bytesToUint16(endianness Endianness, in []byte) (out uint16) {
	switch endianness {
	case BIG_ENDIAN:
		out = binary.BigEndian.Uint16(in)
	case LITTLE_ENDIAN:
		out = binary.LittleEndian.Uint16(in)
	}
	return
}

// Computes the expected length of a modbus RTU response.
func calculateResponseBytes(responseCode uint8, responseLength uint8) (byteCount int, err error) {
	switch responseCode {
	case FCReadHoldingRegisters,
		FCReadInputRegisters,
		FCReadCoils,
		FCReadDiscreteInputs,
		FCReadFileRecord,
		FCWriteFileRecord,
		FCReadWriteMultipleRegisters:
		byteCount = int(responseLength)
	case FCWriteSingleRegister,
		FCWriteMultipleRegisters,
		FCWriteSingleCoil,
		FCWriteMultipleCoils:
		byteCount = 3
	case FCMaskWriteRegister:
		byteCount = 5
	case FCReadHoldingRegisters | 0x80,
		FCReadInputRegisters | 0x80,
		FCReadCoils | 0x80,
		FCReadDiscreteInputs | 0x80,
		FCWriteSingleRegister | 0x80,
		FCWriteMultipleRegisters | 0x80,
		FCWriteSingleCoil | 0x80,
		FCWriteMultipleCoils | 0x80,
		FCMaskWriteRegister | 0x80,
		FCReadFileRecord | 0x80,
		FCWriteFileRecord | 0x80,
		FCReadWriteMultipleRegisters | 0x80,
		FCReadFifoQueue | 0x80:
		byteCount = 0
	case FCReadFifoQueue:
		err = ErrNeedReadMore
	default:
		err = ErrProtocolError
	}
	return
}
