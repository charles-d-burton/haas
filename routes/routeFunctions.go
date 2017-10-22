package routes

import (
	"bytes"
	"fmt"
	"haas/datastores"
	"haas/static"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/goburrow/serial"
)

func StaticHandler(rw http.ResponseWriter, req *http.Request) {
	var path string = req.URL.Path
	if path == "" {
		path = "index.html"
	}
	if bs, err := static.Asset(path); err != nil {
		rw.WriteHeader(http.StatusNotFound)
	} else {
		var reader = bytes.NewBuffer(bs)
		io.Copy(rw, reader)
	}
}

func FileHandler(w http.ResponseWriter, r *http.Request) {

	// the FormFile function takes in the POST input id file
	file, header, err := r.FormFile("file")
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	defer file.Close()
	/*log.Println(formatRequest(r))
	out, err := os.Create("./" + header.Filename)
	if err != nil {
		fmt.Fprintf(w, "Unable to create the file for writing. Check your write access privilege")
		return
	}

	defer out.Close()*/

	out := bytes.NewBuffer(nil)

	// write the content from POST to the file
	_, err = io.Copy(out, file)
	if err != nil {
		fmt.Fprintln(w, err)
	}

	err = datastores.BoltConn.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("gcode"))
		err := b.Put([]byte(header.Filename), out.Bytes())
		return err
	})
	writeToSerial(out.Bytes())
	out.Reset()
	if err != nil {
		log.Println(err)
	}

	fmt.Fprintf(w, "File uploaded successfully : ")
	fmt.Fprintf(w, header.Filename)
}

func writeToSerial(data []byte) {
	config := serial.Config{
		Address:  "/dev/ttyS0",
		BaudRate: 115200,
		DataBits: 8,
		StopBits: 1,
		Parity:   "N",
		Timeout:  120 * time.Second,
	}
	log.Println("Data size: ", len(data))
	log.Printf("connecting %+v", config)
	port, err := serial.Open(&config)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("connected")
	defer func() {
		err := port.Close()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("closed")
	}()

	if _, err = port.Write([]byte(data)); err != nil {
		log.Fatal(err)
	}
	if _, err = io.Copy(os.Stdout, port); err != nil {
		log.Fatal(err)
	}
}

// formatRequest generates ascii representation of a request
func formatRequest(r *http.Request) string {
	// Create return string
	var request []string
	// Add the request string
	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	// Add the host
	request = append(request, fmt.Sprintf("Host: %v", r.Host))
	// Loop through headers
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}

	// If this is a POST, add post data
	if r.Method == "POST" {
		r.ParseForm()
		request = append(request, "\n")
		request = append(request, r.Form.Encode())
	}
	// Return the request as a string
	return strings.Join(request, "\n")
}
