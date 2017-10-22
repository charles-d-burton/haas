package routes

import (
	"bytes"
	"flag"
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

var (
	address  string
	baudrate int
	databits int
	stopbits int
	parity   string
	timeout  int

	message string
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
	err = writeToSerial(out.Bytes())
	out.Reset()
	if err != nil {
		log.Println(err)
		fmt.Fprintf(w, "Problem sending file: ")
		fmt.Fprintf(w, header.Filename)
	}

	fmt.Fprintf(w, "File uploaded successfully : ")
	fmt.Fprintf(w, header.Filename)
}

func writeToSerial(data []byte) error {
	flag.StringVar(&address, "a", "/dev/ttyS0", "address")
	flag.IntVar(&baudrate, "b", 115200, "baud rate")
	flag.IntVar(&databits, "d", 8, "data bits")
	flag.IntVar(&stopbits, "s", 1, "stop bits")
	flag.StringVar(&parity, "p", "N", "parity (N/E/O)")
	flag.StringVar(&message, "m", "serial", "message")
	flag.IntVar(&timeout, "t", 30, "timeout")
	flag.Parse()
	config := serial.Config{
		Address:  address,
		BaudRate: baudrate,
		DataBits: databits,
		StopBits: stopbits,
		Parity:   parity,
		Timeout:  time.Duration(timeout) * time.Second,
	}
	log.Println("Data size: ", len(data))
	log.Printf("connecting %+v", config)
	port, err := serial.Open(&config)
	if err != nil {
		log.Println(err)
		return err
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
		log.Println(err)
		return err
	}
	if _, err = io.Copy(os.Stdout, port); err != nil {
		log.Println(err)
		return err
	}
	return err
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
