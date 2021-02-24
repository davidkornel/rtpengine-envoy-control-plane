package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
)

const maxBufferSize = 1024

type bencodedMessage struct {
	ICE     string
	CallId  string "call-id"
	Command string
	FromTag string "from-tag"
	Label   string
	Sdp     string
}

type Message struct {
	CallerRTP  uint32 `json:"caller_rtp"`
	CallerRTCP uint32 `json:"caller_rtcp"`
	CalleeRTP  uint32 `json:"callee_rtp"`
	CalleeRTCP uint32 `json:"callee_rtcp"`
}

func Server(address string, l *Logger) (err error) {
	// ListenPacket provides us a wrapper around ListenUDP so that
	// we don't need to call `net.ResolveUDPAddr` and then subsequentially
	// perform a `ListenUDP` with the UDP listenerAddress.
	//
	// The returned value (PacketConn) is pretty much the same as the one
	// from ListenUDP (UDPConn) - the only difference is that `Packet*`
	// methods and interfaces are more broad, also covering `ip`.
	pc, err := net.ListenPacket("udp", address)
	if err != nil {
		return
	} else {
		log.Printf("udp server listening on %d\n", 1234)
	}
	ctx := context.Background()

	// `Close`ing the packet "connection" means cleaning the data structures
	// allocated for holding information about the listening socket.
	defer pc.Close()
	doneChan := make(chan error, 1)
	buffer := make([]byte, maxBufferSize)

	// Given that waiting for packets to arrive is blocking by nature and we want
	// to be able of canceling such action if desired, we do that in a separate
	// go routine.
	go func() {
		for {
			// By reading from the connection into the buffer, we block until there's
			// new content in the socket that we're listening for new packets.
			//
			// Whenever new packets arrive, `buffer` gets filled and we can continue
			// the execution.
			//
			// note.: `buffer` is not being reset between runs.
			//	  It's expected that only `n` reads are read from it whenever
			//	  inspecting its contents.
			n, addr, err := pc.ReadFrom(buffer)
			if err != nil {
				doneChan <- err
				return
			}

			fmt.Printf("packet-received: bytes=%d from=%s\n",
				n, addr.String())
			fmt.Printf("packet content should be SDP: %s", buffer[:n])

			//Below line is only for bencoded messages
			//unmarshalBencodedMessage(buffer[:n], l)

			decodeJSON(buffer[:n], l)

		}
	}()

	select {
	case <-ctx.Done():
		fmt.Println("cancelled")
		err = ctx.Err()
	case err = <-doneChan:
	}

	return
}

func decodeJSON(buffer []byte, l *Logger) {
	//TEST
	//const jsonStream = `
	//{"caller_rtp": 10020,"caller_rtcp": 10021, "callee_rtp": 10030, "callee_rtcp": 10031}
	//`
	//TEST OVER
	dec := json.NewDecoder(strings.NewReader(string(buffer[:])))

	var m Message

	err := dec.Decode(&m)
	if err != nil {
		l.Errorf("Error while decoding JSON: %s\n", err)
	} else {
		l.Debugf("CallerRTP: %d, CallerRTCP: %d, CalleeRTP: %d, CalleeRTCP: %d\n", m.CallerRTP, m.CallerRTCP, m.CalleeRTP, m.CalleeRTCP)
		createNewListeners(m)

		updateConfig("ingress", l)
		updateConfig("sidecar", l)
	}
}

/*
//unmarshalBencodedMessage is able to process a bencoded []byte into bencodedMessage struct
func unmarshalBencodedMessage(buffer []byte, l *Logger) {
	var ub = bencodedMessage{"ice", "Call-id", "command",
		"from-tag", "label", "sdp"}
	//fmt.Printf("%s", buffer)
	i := bytes.Index(buffer, []byte(" d"))
	buffer = append(buffer[i+1:])
	fmt.Printf("-------%s-------\n", buffer)
	r := bytes.NewReader(buffer)
	err := bencode.Unmarshal(r, &ub)
	if err == nil {
		fmt.Printf("\nunmarshaled content: %+v\n", ub)
		updateConfig("ingress", l)
		updateConfig("sidecar", l)
	} else {
		fmt.Println(err)
	}
}
*/
