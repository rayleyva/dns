// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// DNS resolver client: see RFC 1035.
// A dns resolver is to be run as a seperate goroutine. 
// For every reply the resolver answers by sending the
// received packet back on the channel.

package dns

import (
	"os"
	"rand"
	"time"
	"net"
)

type MsgErr struct {
	M *Msg
	E os.Error
}

type Resolver struct {
	Servers  []string // servers to use
	rtt      []int    // round trip times for each NS (TODO)
	Search   []string // suffixes to append to local name
	Port     string   // what port to use
	Ndots    int      // number of dots in name to trigger absolute lookup
	Timeout  int      // seconds before giving up on packet
	Attempts int      // lost packets before giving up on server
	Rotate   bool     // round robin among servers
}

// do it
func Query(res *Resolver, msg chan MsgErr, quit chan bool) {
	var c net.Conn
	var err os.Error
	var in *Msg
	for {
		select {
		case <-quit: // quit signal recevied
			println("Quiting")
			// send something back on the channel?
			return
		case out := <-msg: //msg received
			var cerr os.Error
			println("Getting a message")
			// Set an id
			//if len(name) >= 256 {
			out.M.Id = uint16(rand.Int()) ^ uint16(time.Nanoseconds())
			println("Setting the id", out.M.Id)
			sending, ok := out.M.Pack()
			if !ok {
				println("error converting")
				msg <- MsgErr{nil, nil} // todo error
			}
			println("here")

			for i := 0; i < len(res.Servers); i++ {
				println("here", i)
				server := res.Servers[i] + ":53"
				println(server)

				println("before dial")
				c, cerr = net.Dial("udp", "", server)
				println("after dial")
				if cerr != nil {
					println("error sending")
					err = cerr
					continue
				}
				println("exchange")
				in, err = exchange(c, sending, res.Attempts, res.Timeout)
				// Check id in.id != out.id

				c.Close()
				if err != nil {
					println("Err not nil")
					continue
				}
			}
			println("komt ik hier dan")
			if err != nil {
				msg <- MsgErr{nil, err}
			} else {
				msg <- MsgErr{in, nil}
			}
		}
	}
	println("Mag nooit hier komen")
	return
}

// Use Pack to create a DNS question, from a msg

// Send a request on the connection and hope for a reply.
// Up to res.Attempts attempts.
func exchange(c net.Conn, m []byte, attempts, timeout int) (*Msg, os.Error) {
	for attempt := 0; attempt < attempts; attempt++ {
		n, err := c.Write(m)
		if err != nil {
			return nil, err
		}

		c.SetReadTimeout(int64(timeout) * 1e9) // nanoseconds
		// EDNS TODO
		buf := make([]byte, 2000) // More than enough.
		n, err = c.Read(buf)
		if err != nil {
			// More Go foo needed
			//if e, ok := err.(Error); ok && e.Timeout() {
			//	continue
			//}
			return nil, err
		}
		buf = buf[0:n]
		in := new(Msg)
		if !in.Unpack(buf) {
			continue
		}
		return in, nil
	}
	return nil, nil // todo error
}
