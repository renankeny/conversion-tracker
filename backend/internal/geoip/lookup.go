package geoip

import (
	"errors"
	"net"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

type Lookup struct {
	mu     sync.RWMutex
	reader *geoip2.Reader
}

// New abre a base GeoLite2 City. Se path for vazio devolve um Lookup nulo
// que retorna sempre Result vazio (útil em dev sem o .mmdb).
//
// Se o arquivo não existir ou não puder ser aberto, retorna uma instância
// VAZIA mas válida (com reader=nil) e nil de erro. Isso permite que o admin
// chame Reload() depois pra carregar a base sem precisar reiniciar o processo.
// O caller pode chamar Available() pra verificar se a base está carregada.
func New(path string) (*Lookup, error) {
	l := &Lookup{}
	if path == "" {
		return l, nil
	}
	r, err := geoip2.Open(path)
	if err != nil {
		// Arquivo ausente ou corrompido — retorna instância vazia,
		// permitindo que o admin chame Reload() depois.
		return l, nil
	}
	l.reader = r
	return l, nil
}

func (l *Lookup) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.reader == nil {
		return nil
	}
	err := l.reader.Close()
	l.reader = nil
	return err
}

// Available retorna true se a base GeoIP está carregada e pronta pra resolver.
func (l *Lookup) Available() bool {
	if l == nil {
		return false
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.reader != nil
}

// Reload abre a base no path informado e substitui o reader atual atomicamente.
// Útil pra atualizar a base em runtime sem precisar reiniciar o processo.
func (l *Lookup) Reload(path string) error {
	if l == nil {
		return errors.New("lookup nil")
	}
	if path == "" {
		return errors.New("path vazio")
	}
	r, err := geoip2.Open(path)
	if err != nil {
		return err
	}
	l.mu.Lock()
	old := l.reader
	l.reader = r
	l.mu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	return nil
}

type Result struct {
	Country    string
	State      string
	City       string
	PostalCode string
}

func (l *Lookup) Resolve(ip string) (Result, error) {
	if l == nil {
		return Result{}, nil
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return Result{}, errors.New("invalid ip")
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.reader == nil {
		return Result{}, nil
	}
	rec, err := l.reader.City(parsed)
	if err != nil {
		return Result{}, err
	}
	res := Result{
		Country:    rec.Country.IsoCode,
		City:       rec.City.Names["en"],
		PostalCode: rec.Postal.Code,
	}
	if len(rec.Subdivisions) > 0 {
		res.State = rec.Subdivisions[0].IsoCode
	}
	return res, nil
}
