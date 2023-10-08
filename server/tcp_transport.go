package server

import (
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/blacktear23/modbus_gateway/config"
)

var (
	errNeedRetry = errors.New("Need retry")
)

type tcpTransport struct {
	cfg     *config.Backend
	conn    *net.TCPConn
	timeout time.Duration
	lock    sync.RWMutex
	lastTxn uint16
}

func newTcpTransport(cfg *config.Backend) *tcpTransport {
	return &tcpTransport{
		cfg:     cfg,
		timeout: time.Duration(cfg.Timeout) * time.Millisecond,
	}
}

func (tt *tcpTransport) ensureConn() error {
	tt.lock.RLock()
	if tt.conn != nil {
		tt.lock.RUnlock()
		return nil
	}
	tt.lock.RUnlock()
	addr, err := net.ResolveTCPAddr("tcp", tt.cfg.Address)
	if err != nil {
		return err
	}
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return err
	}
	tt.lock.Lock()
	if tt.conn == nil {
		tt.conn = conn
	} else {
		conn.Close()
	}
	tt.lock.Unlock()
	return nil
}

func (tt *tcpTransport) cleanErrorConn() {
	tt.lock.Lock()
	if tt.conn != nil {
		tt.conn.Close()
	}
	tt.conn = nil
	tt.lock.Unlock()
}

func (tt *tcpTransport) ExecuteRequest(req *pdu) (*pdu, error) {
	if err := tt.ensureConn(); err != nil {
		log.Println("Connect backend got error:", err)
		return modbusErrorPdu(req, MErrGWTargetFailedToRespond), nil
	}
	resp, err := tt.executeRequestTCP(req)
	if err != nil && err == errNeedRetry {
		// Retry time
		tt.cleanErrorConn()
		if err := tt.ensureConn(); err != nil {
			log.Println("Connect backend got error:", err)
			return modbusErrorPdu(req, MErrGWTargetFailedToRespond), nil
		}
		rresp, err := tt.executeRequestTCP(req)
		if err != nil {
			return modbusErrorPdu(req, MErrGWPathUnavailable), err
		}
		return rresp, err
	}
	if err != nil {
		return modbusErrorPdu(req, MErrGWPathUnavailable), err
	}
	return resp, err
}

func (tt *tcpTransport) executeRequestTCP(req *pdu) (*pdu, error) {
	var err error
	if tt.timeout > 0 {
		err = tt.conn.SetDeadline(time.Now().Add(tt.timeout))
		if err != nil {
			return nil, err
		}
		defer func(conn net.Conn) {
			conn.SetDeadline(time.Time{})
		}(tt.conn)
	}
	tt.lastTxn++
	_, err = tt.conn.Write(tt.encodeMBAPFrame(tt.lastTxn, req))
	if err != nil && errors.Is(err, syscall.ECONNRESET) {
		return nil, errNeedRetry
	}
	resp, err := tt.readResponse()
	if err != nil && errors.Is(err, syscall.ECONNRESET) {
		return nil, errNeedRetry
	}
	return resp, err
}

func (tt *tcpTransport) Close() error {
	tt.lock.Lock()
	defer tt.lock.Unlock()
	var err error
	if tt.conn != nil {
		err = tt.conn.Close()
		tt.conn = nil
	}
	return err
}

func (tt *tcpTransport) readResponse() (*pdu, error) {
	var (
		resp  *pdu
		vmbap *mbap
		err   error
	)
	for {
		resp, vmbap, err = tt.readMBAPFrame()
		if err == ErrInvalidProtocol {
			continue
		}

		if err != nil {
			return nil, err
		}

		if vmbap.txnID != tt.lastTxn {
			log.Printf("Receive unexpected transaction id (expected: 0x%04x, got: 0x%04x)", tt.lastTxn, vmbap.txnID)
			continue
		}
		break
	}
	return resp, err
}

func (tt *tcpTransport) readMBAPFrame() (*pdu, *mbap, error) {
	buf := make([]byte, mbapHeaderLen)
	_, err := io.ReadFull(tt.conn, buf)
	if err != nil {
		return nil, nil, err
	}

	txnID := bytesToUint16(BIG_ENDIAN, buf[0:2])
	protocolID := bytesToUint16(BIG_ENDIAN, buf[2:4])
	unitID := buf[6]
	restBytes := int(bytesToUint16(BIG_ENDIAN, buf[4:6]))
	restBytes--
	if restBytes+mbapHeaderLen > maxTCPFrameLen {
		return nil, nil, ErrInvalidProtocol
	}
	if restBytes <= 0 {
		return nil, nil, ErrInvalidProtocol
	}

	pduBuf := make([]byte, restBytes)
	_, err = io.ReadFull(tt.conn, pduBuf)
	if err != nil {
		return nil, nil, err
	}

	if protocolID != 0x0000 {
		log.Printf("Receive unexpected protocol id 0x%04x", protocolID)
		return nil, nil, ErrInvalidProtocol
	}

	vpdu := &pdu{
		unitID:   unitID,
		funcCode: pduBuf[0],
		payload:  pduBuf[1:],
	}
	vmbap := &mbap{
		txnID:      txnID,
		protocolID: protocolID,
		unitID:     unitID,
	}
	return vpdu, vmbap, nil

}

func (tt *tcpTransport) encodeMBAPFrame(txnID uint16, req *pdu) []byte {
	data := uint16ToBytes(BIG_ENDIAN, txnID)
	// Protocol ID 0x0000
	data = append(data, 0x00, 0x00)
	// Length
	data = append(data, uint16ToBytes(BIG_ENDIAN, uint16(2+len(req.payload)))...)
	// Unit ID
	data = append(data, req.unitID)
	// Function Code
	data = append(data, req.funcCode)
	// Payload
	data = append(data, req.payload...)
	return data
}
