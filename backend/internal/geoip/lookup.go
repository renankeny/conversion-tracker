package geoip

import (
	"errors"
	"net"

	"github.com/oschwald/geoip2-golang"
)

type Lookup struct {
	reader *geoip2.Reader
}

// New abre a base GeoLite2 City. Se path for vazio devolve um Lookup nulo
// que retorna sempre Result vazio (útil em dev sem o .mmdb).
func New(path string) (*Lookup, error) {
	if path == "" {
		return &Lookup{}, nil
	}
	r, err := geoip2.Open(path)
	if err != nil {
		return nil, err
	}
	return &Lookup{reader: r}, nil
}

func (l *Lookup) Close() error {
	if l == nil || l.reader == nil {
		return nil
	}
	return l.reader.Close()
}

type Result struct {
	Country    string
	State      string
	City       string
	PostalCode string
}

func (l *Lookup) Resolve(ip string) (Result, error) {
	if l == nil || l.reader == nil {
		return Result{}, nil
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return Result{}, errors.New("invalid ip")
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
