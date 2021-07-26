package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type eventHandler struct {
	counter         uint64
	countEvent      chan uint64
	getCounterEvent chan chan uint64
	resetEvent      chan struct{}
}

func (h *eventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	if r.Method == http.MethodGet {
		c := h.getCounter()
		fmt.Fprintf(w, "Counter: %d\n", c)
	} else if r.Method == http.MethodPut {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "can't parse input")
		} else {
			addition, err := strconv.ParseUint(string(body), 10, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, "wrong input")
			} else {
				h.triggerCounterEvent(addition)
				fmt.Fprintf(w, "Added %d", addition)
			}
		}
	} else if r.Method == http.MethodPost && r.URL.Path == "/reset" {
		h.reset()
		w.WriteHeader(http.StatusNoContent)
	}
}

func newEventHandler() *eventHandler {
	h := &eventHandler{
		counter:         0,
		countEvent:      make(chan uint64, 1),
		getCounterEvent: make(chan chan uint64),
		resetEvent:      make(chan struct{}),
	}

	go h.handle()

	return h
}

var handler = newEventHandler()

func (h *eventHandler) handle() {
	for {
		select {
		case addition := <-h.countEvent:
			h.counter += addition
		case ch := <-h.getCounterEvent:
			ch <- h.counter
		case <-h.resetEvent:
			h.counter = 0
		}

	}
}

// counterEvent wrapper
func (h *eventHandler) triggerCounterEvent(addition uint64) {
	h.countEvent <- addition
}

// getCounterEvent wrapper
func (h eventHandler) getCounter() uint64 {
	callback := make(chan uint64)
	defer close(callback)

	h.getCounterEvent <- callback
	res := <-callback
	return res
}

func (h *eventHandler) reset() {
	h.resetEvent <- struct{}{}
}

func main() {
	http.Handle("/", handler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
