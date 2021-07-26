package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// the event handler type
type eventHandler struct {
	// data is the server data
	data uint64

	// channel to get addition event. The content of the event is a uint64
	// value to be added to the data
	additionEvent chan uint64

	// channel of uint64 channels, to receive the current value of data. The
	// content of the event is a callback channel to return the value of data.
	getEvent chan chan uint64

	// channel to receive reset events. the content is ignored
	resetEvent chan struct{}
}

// main HTTP handler
func (h *eventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	if r.Method == http.MethodGet {
		h.getDataRequest(w, r)
	} else if r.Method == http.MethodPut {
		h.updateDataRequest(w, r)
	} else if r.Method == http.MethodPost && r.URL.Path == "/reset" {
		h.resetDataRequest(w, r)
	}
}

// HTTP handler for the update data request
func (h *eventHandler) updateDataRequest(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintln(w, "can't parse input")
	} else {
		addition, err := strconv.ParseUint(string(body), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintln(w, "wrong input")
		} else {
			h.addToData(addition)
			_, _ = fmt.Fprintf(w, "Added %d\n", addition)
		}
	}
}

// HTTP handler for the get data request
func (h eventHandler) getDataRequest(w http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprintf(w, "Data: %d\n", h.getData())
}

// http handler for the reset data request
func (h eventHandler) resetDataRequest(w http.ResponseWriter, _ *http.Request) {
	h.reset()
	w.WriteHeader(http.StatusNoContent)
}

// create new event handler and start the event handling loop in a new go-routine
func newEventHandler() *eventHandler {
	h := &eventHandler{
		data:          0,
		additionEvent: make(chan uint64, 1),
		getEvent:      make(chan chan uint64, 1),
		resetEvent:    make(chan struct{}, 1),
	}

	go h.handle()

	return h
}

// the event handler instance
var handler = newEventHandler()

// the event handler implementation: use select and channels to implement locking
// because only one event is handled in a given time.
func (h *eventHandler) handle() {
	for {
		select {
		case addition := <-h.additionEvent:
			h.data += addition
		case ch := <-h.getEvent:
			ch <- h.data
		case <-h.resetEvent:
			h.data = 0
		}

	}
}

// additionEvent wrapper
func (h *eventHandler) addToData(addition uint64) {
	h.additionEvent <- addition
}

// getEvent wrapper. Use a callback channel to return the response.
func (h eventHandler) getData() uint64 {
	callback := make(chan uint64)
	defer close(callback)

	h.getEvent <- callback
	res := <-callback
	return res
}

// reset data wrapper
func (h *eventHandler) reset() {
	h.resetEvent <- struct{}{}
}

// main: run the server on port 8080
func main() {
	http.Handle("/", handler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
