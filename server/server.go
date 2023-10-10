package server

import (
	"errors"
	"io"
	"log"
	"net"
	"time"
)

const (
	mbapHeaderLen  = 7
	maxTCPFrameLen = 260
)

var (
	ErrInvalidProtocol = errors.New("Invalid Protocol")
)

type TCPServer struct {
	running bool
	listen  string
	router  *Router
	ln      net.Listener
	timeout time.Duration
}

func NewTCPServer(listen string, timeout int, router *Router) *TCPServer {
	return &TCPServer{
		listen:  listen,
		router:  router,
		timeout: time.Duration(timeout) * time.Second,
	}
}

func (s *TCPServer) Start() error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", s.listen)
	if err != nil {
		return err
	}
	s.ln, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}
	s.running = true
	go s.runListen()
	return nil
}

func (s *TCPServer) Stop() error {
	s.running = false
	if s.ln != nil {
		return s.ln.Close()
	}
	return nil
}

func (s *TCPServer) runListen() {
	for s.running {
		conn, err := s.ln.Accept()
		if err != nil {
			if s.running {
				log.Printf("Accept Error: %v", err)
			}
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *TCPServer) handleConn(conn net.Conn) {
	defer conn.Close()
	for {
		// Read the request
		req, pdu, err := s.readRequest(conn)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Println("Read request got error:", err)
			}
			break
		}
		// Route to backend
		uid := pdu.unitID
		resp, err := s.routeRequest(uid, pdu)
		if err != nil {
			log.Println("Get response got error:", err)
		}
		// Error will report and resp PDU will set to error
		// So check resp is nil if yes break
		if resp == nil {
			break
		}
		// Write the response
		err = s.writeResponse(conn, req, resp)
		if err != nil {
			log.Println("Write response got error:", err)
			break
		}
	}
}

func (s *TCPServer) readRequest(conn net.Conn) (*mbap, *pdu, error) {
	buf := make([]byte, mbapHeaderLen)
	_, err := io.ReadFull(conn, buf)
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
	_, err = io.ReadFull(conn, pduBuf)
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
	return vmbap, vpdu, nil
}

func (s *TCPServer) writeResponse(conn net.Conn, req *mbap, resp *pdu) error {
	length := uint16(len(resp.payload) + 2)
	buf := make([]byte, 0, maxTCPFrameLen)
	// Transaction ID 2 bytes
	buf = append(buf, uint16ToBytes(BIG_ENDIAN, req.txnID)...)
	// Protocol ID 2 bytes
	buf = append(buf, uint16ToBytes(BIG_ENDIAN, req.protocolID)...)
	// Length 2 bytes
	buf = append(buf, uint16ToBytes(BIG_ENDIAN, length)...)
	// Unit ID from request 1 bytes
	buf = append(buf, req.unitID)
	// Function code 1 byte
	buf = append(buf, resp.funcCode)
	// Payload N bytes
	buf = append(buf, resp.payload...)

	// Send data to client
	_, err := conn.Write(buf)
	return err
}

func (s *TCPServer) routeRequest(uid uint8, req *pdu) (*pdu, error) {
	return s.router.RequestBackend(uid, req)
}
