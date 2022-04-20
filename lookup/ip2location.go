package lookup

import (
	"errors"
	"log"
	"net"
	"os"
	"strings"

	"github.com/ip2location/ip2location-go/v9"
)

type i2l struct {
	db *ip2location.DB
}

// OpenIP2location opens a ip2location database and returns a Lookup.
func OpenIP2location(dbname string) (Lookup, error) {
	db, err := ip2location.OpenDB(dbname)
	if os.IsNotExist(err) {
		log.Printf("%s not found, fallback on static asset", dbname)

		var payload []byte
		payload, err = database(dbname)
		if err != nil {
			return nil, err
		}

		db, err = ip2location.OpenDBWithReader(newrcat(payload))
	}

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
