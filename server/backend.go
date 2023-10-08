package server

import (
	"errors"
	"log"

	"github.com/blacktear23/modbus_gateway/config"
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
	trans   Transport
	running bool
	ch      chan *modbusRequest
}

func NewBackend(cfg *config.Backend) *Backend {
	return &Backend{
		Name:  cfg.Name,
		bcfg:  cfg,
		trans: newTransport(cfg),
		ch:    make(chan *modbusRequest, 1),
	}
}

func (b *Backend) GetBackendKey() string {
	return b.bcfg.GetBackendKey()
}

func (b *Backend) Stop() error {
	b.running = false
	close(b.ch)
	err := b.trans.Close()
	log.Printf("Stop backend %s", b.Name)
	return err
}

func (b *Backend) Start() {
	b.running = true
	go b.start()
}

func (b *Backend) start() {
	log.Printf("Start running backend %s", b.Name)
	for req := range b.ch {
		respPdu, err := b.trans.ExecuteRequest(req.req)
		mresp := &modbusResponse{
			resp: respPdu,
			err:  err,
		}
		req.respCh <- mresp
	}
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
	b.ch <- mreq
	resp, ok := <-respCh
	if !ok {
		return nil, errors.New("Client closed")
	}
	return resp.resp, resp.err
}
