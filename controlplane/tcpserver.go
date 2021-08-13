package controlplane

import (
	"encoding/json"
	"fmt"
	"net"
)

//const maxBufferSize = 1024

/*type bencodedMessage struct {
	ICE     string
	CallId  string "call-id"
	Command string
	FromTag string "from-tag"
	Label   string
	Sdp     string
}*/

type Message struct {
	CallerRTP   uint32 `json:"caller_rtp"`
	CallerRTCP  uint32 `json:"caller_rtcp"`
	CalleeRTP   uint32 `json:"callee_rtp"`
	CalleeRTCP  uint32 `json:"callee_rtcp"`
	CallId      string `json:"call_id"`
	RtpeAddress string `json:"rtpe_address"`
}

func Server(tcpPort uint, l *Logger) (err error) {
	ltcp, err := net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", tcpPort))
	if err != nil {
		fmt.Println(err)
		return
	} else {
		l.Infof("tcp server listening on %d\n", tcpPort)
	}
	defer func(ltcp net.Listener) {
		err := ltcp.Close()
		if err != nil {
			return
		}
	}(ltcp)

	for {
		c, err := ltcp.Accept()
		if err != nil {
			fmt.Println(err)
			return err
		}
		go handleConnection(c, l)
	}
	//return
}

func handleConnection(c net.Conn, l *Logger) {
	fmt.Printf("Serving %s\n", c.RemoteAddr().String())
	for {
		e := decodeJSON(c, l)
		if e != nil {
			l.Errorf(`DecodeJSON returned error: `, e)
		}
	}
	//err := c.Close()
	//if err != nil {
	//	return
	//}
}

func decodeJSON(c net.Conn, l *Logger) (error error) {
	//TEST
	//const jsonStream = `
	//{"caller_rtp": 10020,"caller_rtcp": 10021, "callee_rtp": 10030, "callee_rtcp": 10031}
	//`
	//TEST OVER
	//dec := json.NewDecoder(strings.NewReader(string(buffer[:])))
	dec := json.NewDecoder(c)
	var m Message

	err := dec.Decode(&m)
	if err != nil {
		l.Errorf("Error while decoding JSON: %s\n", err)
		return err
	} else {
		l.Debugf("CallerRTP: %d, CallerRTCP: %d, CalleeRTP: %d, CalleeRTCP: %d\n", m.CallerRTP, m.CallerRTCP, m.CalleeRTP, m.CalleeRTCP)
		createNewListeners(m)

		updateConfig("ingress", l)
		updateConfig("sidecar", l)
	}
	return nil
}
