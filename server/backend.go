package server

import (
	"errors"
	"log"

	"github.com/blacktear23/modbus_gateway/config"
)

var (
	ErrClientClosed = errors.New("Client closed")
)

type Transport interface {
	ExecuteRequest(req *pdu) (*pdu, error)
	Close() error
}

type modbusRequest struct {
	req    *pdu
	respCh chan *modbusResponse
}

type modbusResponse struct {
	resp *pdu
	err  error
}

type Backend struct {
	Name    string
	bcfg    *config.Backend
	trans   []Transport
	ch      chan *modbusRequest
	running bool
}

func NewBackend(cfg *config.Backend) *Backend {
	transports := newTransports(cfg)
	return &Backend{
		Name:  cfg.Name,
		bcfg:  cfg,
		trans: transports,
		ch:    make(chan *modbusRequest, len(transports)),
	}
}

func (b *Backend) GetBackendKey() string {
	return b.bcfg.GetBackendKey()
}

func (b *Backend) Stop() error {
	var err error
	b.running = false
	close(b.ch)
	for i, trans := range b.trans {
		ierr := trans.Close()
		if ierr != nil {
			log.Printf("Close backend transport %s [%d] got error: %v", b.Name, i, ierr)
			err = ierr
		}
	}
	log.Printf("Stop backend %s", b.Name)
	return err
}

func (b *Backend) Start() {
	b.running = true
	for i, _ := range b.trans {
		go b.start(i)
	}
}

func (b *Backend) start(idx int) {
	log.Printf("Start running backend transport %s [%d]", b.Name, idx)
	for req := range b.ch {
		respPdu, err := b.trans[idx].ExecuteRequest(req.req)
		mresp := &modbusResponse{
			resp: respPdu,
			err:  err,
		}
		req.respCh <- mresp
	}
}

func (b *Backend) safeSend(req *modbusRequest) (err error) {
	defer func() {
		if recover() != nil {
			err = ErrClientClosed
		}
	}()

	b.ch <- req
	return
}

func (b *Backend) ExecuteRequest(req *pdu) (*pdu, error) {
	if !b.running {
		return nil, errors.New("Backend not running")
	}

	respCh := make(chan *modbusResponse, 1)
	defer func(ch chan *modbusResponse) {
		close(ch)
	}(respCh)

	mreq := &modbusRequest{
		req:    req,
		respCh: respCh,
	}

	err := b.safeSend(mreq)
	if err != nil {
		return nil, err
	}

	resp, ok := <-respCh
	if !ok {
		return nil, ErrClientClosed
	}
	return resp.resp, resp.err
}
