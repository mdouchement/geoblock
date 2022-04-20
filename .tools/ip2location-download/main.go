package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/ip2location/ip2location-go/v9"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("invalid arguments number")
	}

	url := os.Args[1]
	filename := os.Args[2]

	err := download(url, filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = verify(filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func download(url, filename string) error {
	res, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("downloading db failed: %w", err)
	}
	defer res.Body.Close()

	payload, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("downloading db failed: %w", err)
	}

	var r io.Reader = bytes.NewReader(payload)
	if strings.HasSuffix(strings.ToLower(url), ".zip") {
		r = nil

		codec, err := zip.NewReader(bytes.NewReader(payload), res.ContentLength)
		if err != nil {
			return fmt.Errorf("creating zip reader: %w", err)
		}

		for _, file := range codec.File {
			if !strings.HasSuffix(strings.ToLower(file.Name), ".bin") {
				continue
			}

			f, err := file.Open()
			if err != nil {
				return fmt.Errorf("opening zip db file: %w", err)
			}
			defer f.Close()

			r = f
		}
	}

	if r == nil {
		return errors.New("db file not found in downloaded archive")
	}

	//

	w, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer w.Close()

	_, err = io.Copy(w, r)
	if err != nil {
		return fmt.Errorf("copy db to file: %w", err)
	}

	return w.Sync()
}

func verify(filename string) error {
	db, err := ip2location.OpenDB(filename)
	if err != nil {
		return fmt.Errorf("opening db failed: %w", err)
	}
	defer db.Close()

	rec, err := db.Get_country_short("1.1.1.1")
	if err != nil {
		return fmt.Errorf("querying db failed: %w", err)
	}

	if rec.Country_short != "US" {
		return fmt.Errorf("query returned unexpected result, db is likely corrupted")
	}

	return nil
}
