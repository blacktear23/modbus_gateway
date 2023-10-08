package server

import (
	"time"

	"github.com/blacktear23/modbus_gateway/config"
	"github.com/goburrow/serial"
)

type SerialConn struct {
	cfg  *config.Backend
	port serial.Port
}

func newSerialConn(cfg *config.Backend) *SerialConn {
	return &SerialConn{
		cfg: cfg,
	}
}

func (c *SerialConn) Open() error {
	var err error
	c.port, err = serial.Open(&serial.Config{
		Address:  c.cfg.Address,
		BaudRate: c.cfg.Baudrate,
		DataBits: c.cfg.Databits,
		StopBits: c.cfg.Stopbits,
		Parity:   c.cfg.Parity,
		Timeout:  time.Duration(c.cfg.Timeout) * time.Millisecond,
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *SerialConn) Close() error {
	if c.port == nil {
		return nil
	}
	return c.port.Close()
}

func (c *SerialConn) Read(buf []byte) (int, error) {
	return c.port.Read(buf)
}

func (c *SerialConn) Write(buf []byte) (int, error) {
	return c.port.Write(buf)
}
