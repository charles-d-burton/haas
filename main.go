package main

import (
	"fmt"
	"haas/datastores"
	"haas/routes"
	"log"
	"net/http"

	"github.com/boltdb/bolt"
)

//go:generate go-bindata -prefix "static/" -pkg static -o ./static/bindata.go static/...

func main() {
	// Open the my.db data file in your current directory.
	// It will be created if it doesn't exist.
	var err error
	datastores.BoltConn, err = bolt.Open("gcode.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	initBolt(datastores.BoltConn)
	defer datastores.BoltConn.Close()
	http.Handle("/", http.StripPrefix("/", http.HandlerFunc(routes.StaticHandler)))
	http.Handle("/receive", http.StripPrefix("/", http.HandlerFunc(routes.FileHandler)))
	http.ListenAndServe(":3000", nil)
}

func initBolt(db *bolt.DB) {
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("gcode"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
}
