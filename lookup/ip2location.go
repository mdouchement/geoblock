package lookup

import (
	"errors"
	"io"
	"net"
	"strings"

	"github.com/ip2location/ip2location-go/v9"
)

// A Reader is used to load databases.
type Reader struct {
	io.ReadCloser
	io.ReaderAt
}

type i2l struct {
	db *ip2location.DB
}

// OpenIP2location opens an ip2location database and returns a Lookup.
func OpenIP2location(dbname string) (Lookup, error) {
	db, err := ip2location.OpenDB(dbname)

	return &i2l{
		db: db,
	}, err
}

// OpenIP2locationReader reads an ip2location database and returns a Lookup.
func OpenIP2locationReader(r Reader) (Lookup, error) {
	db, err := ip2location.OpenDBWithReader(r)

	return &i2l{
		db: db,
	}, err
}

func (l *i2l) Country(ip net.IP) (string, error) {
	record, err := l.db.Get_country_short(ip.String())
	if err != nil {
		return "", err
	}

	country := strings.ToLower(record.Country_short)
	if strings.HasPrefix(country, "invalid") {
		return "", errors.New(country)
	}

	if country == "-" {
		return PrivateAddress, nil
	}

	return country, nil
}
