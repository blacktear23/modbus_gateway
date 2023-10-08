package server

import (
	"io"
	"log"
	"sync"
	"time"

	"github.com/blacktear23/modbus_gateway/config"
)

const (
	maxRTUFrameLength = 256
)

type serialTransport struct {
	cfg          *config.Backend
	conn         *SerialConn
	lastActivity time.Time
	t35          time.Duration
	t1           time.Duration
	lock         sync.RWMutex
}

func newSerialTransport(cfg *config.Backend) *serialTransport {
	var t35 time.Duration
	if cfg.Baudrate >= 19200 {
		t35 = 1750 * time.Microsecond
	} else {
		t35 = (serialCharTime(cfg.Baudrate) * 35) / 10
	}
	return &serialTransport{
		cfg: cfg,
		t1:  serialCharTime(cfg.Baudrate),
		t35: t35,
	}
}

func (st *serialTransport) ensureConn() error {
	st.lock.RLock()
	if st.conn != nil {
		st.lock.RUnlock()
		return nil
	}
	st.lock.RUnlock()
	conn := newSerialConn(st.cfg)
	err := conn.Open()
	if err != nil {
		return err
	}
	st.lock.Lock()
	if st.conn == nil {
		st.conn = conn
	} else {
		conn.Close()
	}
	st.lock.Unlock()
	return nil
}

func (st *serialTransport) ExecuteRequest(req *pdu) (*pdu, error) {
	if err := st.ensureConn(); err != nil {
		log.Println("Connect backend got error:", err)
		return modbusErrorPdu(req, MErrGWTargetFailedToRespond), nil
	}
	return st.executeRequestRTU(req)
}

// it will always return pdu response
func (st *serialTransport) executeRequestRTU(req *pdu) (*pdu, error) {
	// if the line was active less than 3.5 char times ago,
	// let t3.5 expire before transmitting
	t := time.Since(st.lastActivity.Add(st.t35))
	if t < 0 {
		time.Sleep(t * (-1))
	}

	ts := time.Now()
	// build an RTU ADU out of the request object and
	// send the final ADU+CRC on the wire
	n, err := st.conn.Write(st.encodeRTUFrame(req))
	if err != nil {
		return modbusErrorPdu(req, MErrGWPathUnavailable), err
	}

	// estimate how long the serial line was busy for.
	// note that on most platforms, Write() will be buffered and return
	// immediately rather than block until the buffer is drained
	st.lastActivity = ts.Add(time.Duration(n) * st.t1)

	// observe inter-frame delays
	time.Sleep(st.lastActivity.Add(st.t35).Sub(time.Now()))

	// read the response back from the wire
	resp, err := st.readRTUFrame()

	if err == ErrBadCRC || err == ErrProtocolError || err == ErrShortFrame {
		// wait for and flush any data coming off the link to allow
		// devices to re-sync
		time.Sleep(time.Duration(maxRTUFrameLength) * st.t1)
		discard(st.conn)
	}

	// mark the time if we heard anything back
	st.lastActivity = time.Now()

	if err != nil {
		return modbusErrorPdu(req, MErrGWPathUnavailable), err
	}

	return resp, err
}

func (st *serialTransport) readRTUFrame() (*pdu, error) {
	buf := make([]byte, maxRTUFrameLength)

	n, err := io.ReadFull(st.conn, buf[0:3])
	if (n > 0 || err == nil) && n != 3 {
		return nil, ErrShortFrame
	}

	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, err
	}

	startPos := 3
	restBytes, err := calculateResponseBytes(uint8(buf[1]), uint8(buf[2]))
	if err != nil {
		if err == ErrNeedReadMore {
			// Read one more byte
			n, err := io.ReadFull(st.conn, buf[3:4])
			if (n > 0 || err == nil) && n != 1 {
				return nil, ErrShortFrame
			}
			if err != nil && err != io.ErrUnexpectedEOF {
				return nil, err
			}
			restBytes = int(bytesToUint16(BIG_ENDIAN, buf[2:4]))
			startPos++
		} else {
			return nil, err
		}
	}
	// Add for CRC
	restBytes += 2

	if n+restBytes > maxRTUFrameLength {
		return nil, ErrProtocolError
	}

	n, err = io.ReadFull(st.conn, buf[startPos:startPos+restBytes])
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, err
	}

	if n != restBytes {
		return nil, ErrShortFrame
	}

	var crc crc
	crc.init()
	crc.add(buf[0 : startPos+restBytes-2])
	if !crc.isEqual(buf[startPos+restBytes-2], buf[startPos+restBytes-1]) {
		return nil, ErrBadCRC
	}
	resp := &pdu{
		unitID:   buf[0],
		funcCode: buf[1],
		payload:  buf[2 : startPos+restBytes-2],
	}
	return resp, nil
}

func (st *serialTransport) encodeRTUFrame(req *pdu) []byte {
	var (
		crc crc
		adu []byte
	)
	adu = append(adu, req.unitID)
	adu = append(adu, req.funcCode)
	adu = append(adu, req.payload...)
	// calculate crc
	crc.init()
	crc.add(adu)
	adu = append(adu, crc.value()...)
	return adu
}

func (st *serialTransport) Close() error {
	st.lock.Lock()
	defer st.lock.Unlock()
	var err error
	if st.conn != nil {
		err = st.conn.Close()
		st.conn = nil
	}
	return err
}

// Returns how long it takes to send 1 byte on a serial line at the
// specified baud rate.
func serialCharTime(rate_bps int) (ct time.Duration) {
	// note: an RTU byte on the wire is:
	// - 1 start bit,
	// - 8 data bits,
	// - 1 parity or stop bit
	// - 1 stop bit
	ct = (11) * time.Second / time.Duration(rate_bps)
	return
}

// Discards the contents of the link's rx buffer, eating up to 1kB of data.
// Note that on a serial line, this call may block for up to serialConf.Timeout
// i.e. 10ms.
func discard(conn *SerialConn) {
	var rxbuf = make([]byte, 1024)
	io.ReadFull(conn, rxbuf)
	return
}
