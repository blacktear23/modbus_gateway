package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"sync"

	"gopkg.in/yaml.v2"
)

var (
	ErrRequireBackendName    = errors.New("Require backend name field")
	ErrRequireBackendAddress = errors.New("Require backend address field")
	ErrInvalidUnitID         = errors.New("Invalid Unit ID")
)

type UnitMap struct {
	UnitID       int    `yaml:"unit_id"`
	Backend      string `yaml:"backend"`
	TargetUnitID int    `yaml:"target_unit_id"`
}

func (u *UnitMap) Validate() error {
	if u.UnitID < 1 || u.UnitID > 255 {
		return ErrInvalidUnitID
	}
	if u.TargetUnitID < 1 || u.TargetUnitID > 255 {
		return ErrInvalidUnitID
	}
	return nil
}

type Backend struct {
	Name     string `yaml:"name"`
	Protocol string `yaml:"protocol"`
	Address  string `yaml:"address"`
	Baudrate int    `yaml:"baudrate"`
	Databits int    `yaml:"databits"`
	Stopbits int    `yaml:"stopbits"`
	Parity   string `yaml:"parity"`
	Timeout  int    `yaml:"timeout"`
}

func (b *Backend) FillDefaults() {
	if b.Protocol == "serial" {
		b.fillSerialDefaults()
	}
}

func (b *Backend) fillSerialDefaults() {
	if b.Baudrate == 0 {
		b.Baudrate = 9600
	}
	if b.Databits == 0 {
		b.Databits = 8
	}
	if b.Stopbits == 0 {
		b.Stopbits = 1
	}
	if b.Parity == "" {
		b.Parity = "N"
	}
}

func (b *Backend) GetBackendKey() string {
	base := fmt.Sprintf("%s %s %s %d", b.Name, b.Protocol, b.Address, b.Timeout)
	if b.Protocol == "tcp" {
		return base
	}
	serialKey := fmt.Sprintf(" %d %d %d %s", b.Baudrate, b.Databits, b.Stopbits, b.Parity)
	return base + serialKey
}

func (b *Backend) Validate() error {
	b.FillDefaults()
	if b.Name == "" {
		return ErrRequireBackendName
	}
	if b.Address == "" {
		return ErrRequireBackendAddress
	}
	switch b.Protocol {
	case "tcp", "serial":
	default:
		return fmt.Errorf("Invalid protocol %s", b.Protocol)
	}
	switch b.Protocol {
	case "tcp":
		return b.validateTcp()
	case "serial":
		return b.validateSerial()
	}
	return nil
}

func (b *Backend) validateTcp() error {
	_, err := net.ResolveTCPAddr("tcp", b.Address)
	return err
}

func (b *Backend) validateSerial() error {
	switch b.Parity {
	case "N", "E", "O":
	default:
		return fmt.Errorf("Invalid parity option: %s", b.Parity)
	}
	return nil
}

type Config struct {
	fname           string
	lock            sync.RWMutex
	Backends        []*Backend `yaml:"backends"`
	UnitMaps        []*UnitMap `yaml:"unit_map"`
	unitIDToBackend map[uint8]*Backend
	unitIDToUnitMap map[uint8]*UnitMap
	backendByName   map[string]*Backend
}

func NewConfig(fname string) (*Config, error) {
	cfg := &Config{
		fname: fname,
	}
	err := cfg.Reload()
	return cfg, err
}

func (c *Config) Reload() error {
	data, err := ioutil.ReadFile(c.fname)
	if err != nil {
		return err
	}

	nc := &Config{}
	if err = yaml.Unmarshal(data, nc); err != nil {
		return err
	}

	backendByName := map[string]*Backend{}

	for _, b := range nc.Backends {
		if err = b.Validate(); err != nil {
			return err
		}
		name := b.Name
		if _, have := backendByName[name]; have {
			return fmt.Errorf("Backend name %s is duplicate", name)
		} else {
			backendByName[name] = b
		}
	}
	uidToBackend := map[uint8]*Backend{}
	uidToUMap := map[uint8]*UnitMap{}

	for _, um := range nc.UnitMaps {
		if err = um.Validate(); err != nil {
			return err
		}
		bname := um.Backend
		backend, have := backendByName[bname]
		if !have {
			return fmt.Errorf("Cannot find backend %s", bname)
		}
		uid := uint8(um.UnitID)
		uidToBackend[uid] = backend
		uidToUMap[uid] = um
	}

	c.lock.Lock()
	c.Backends = nc.Backends
	c.UnitMaps = nc.UnitMaps
	c.unitIDToBackend = uidToBackend
	c.unitIDToUnitMap = uidToUMap
	c.backendByName = backendByName
	c.lock.Unlock()
	return nil
}

func (c *Config) GetBackends() []*Backend {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.Backends
}

func (c *Config) GetBackendByName(name string) *Backend {
	c.lock.RLock()
	back, have := c.backendByName[name]
	c.lock.RUnlock()
	if have {
		return back
	}
	return nil
}

func (c *Config) GetUnitIDMap(uid uint8) (*UnitMap, *Backend) {
	c.lock.RLock()
	umap, uhave := c.unitIDToUnitMap[uid]
	back, bhave := c.unitIDToBackend[uid]
	c.lock.RUnlock()
	if uhave && bhave {
		return umap, back
	}
	return nil, nil
}
