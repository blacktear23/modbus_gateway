package server

import (
	"log"
	"sync"

	"github.com/blacktear23/modbus_gateway/config"
)

type Router struct {
	cfg      *config.Config
	backends map[string]*Backend
	lock     sync.RWMutex
}

func NewRouter(cfg *config.Config) *Router {
	return &Router{
		cfg:      cfg,
		backends: map[string]*Backend{},
	}
}

func (r *Router) GetBackendByUnitID(uid uint8) (*Backend, uint8) {
	umap, back := r.cfg.GetUnitIDMap(uid)
	if umap == nil || back == nil {
		return nil, 0
	}
	backend := r.getBackend(umap.Backend)
	if backend == nil {
		return nil, 0
	}
	return backend, uint8(umap.TargetUnitID)
}

func (r *Router) RequestBackend(uid uint8, req *pdu) (*pdu, error) {
	backend, tuid := r.GetBackendByUnitID(uid)
	// No background target
	if backend == nil {
		return r.respModbusError(uid, req, MErrGWTargetFailedToRespond), nil
	}
	// Transform to target unit ID
	req.unitID = tuid
	resp, err := backend.ExecuteRequest(req)
	// Restore unit ID to origin
	if resp != nil {
		resp.unitID = uid
	}
	return resp, err
}

func (r *Router) respModbusError(uid uint8, req *pdu, errCode uint8) *pdu {
	eresp := &pdu{
		unitID:   uid,
		funcCode: (0x80 | req.funcCode),
		payload:  []byte{errCode},
	}
	return eresp
}

func (r *Router) getBackend(name string) *Backend {
	r.lock.RLock()
	backend, have := r.backends[name]
	r.lock.RUnlock()
	if have {
		return backend
	}
	return r.createNewBackend(name)
}

func (r *Router) createNewBackend(name string) *Backend {
	bcfg := r.cfg.GetBackendByName(name)
	if bcfg == nil {
		return nil
	}
	backend := NewBackend(bcfg)
	r.lock.Lock()
	r.backends[name] = backend
	r.lock.Unlock()
	backend.Start()
	return backend
}

func (r *Router) Reload() {
	r.lock.Lock()
	defer r.lock.Unlock()
	for _, b := range r.backends {
		err := b.Stop()
		if err != nil {
			log.Printf("Close backend %s got error: %v", b.bcfg.Name, err)
		}
	}
	r.backends = map[string]*Backend{}
}
