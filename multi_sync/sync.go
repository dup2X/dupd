// Just a demo for mutil idc sync
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

type Message struct {
	Key, Val string
}

type Proxy struct {
	addr     string
	others   struct{}
	syncNode *Node
}

func NewProxy(readAddr string, peers []string) *Proxy {
	node := &Node{
		id:        readAddr,
		in:        make(chan *Message, 1024),
		out:       make(chan *Message, 1024),
		reader:    NewHTTPReader(readAddr),
		broadcast: newBroadcast(peers),
		wg:        new(sync.WaitGroup),
	}
	return &Proxy{
		addr:     readAddr,
		syncNode: node,
	}
}

func (p *Proxy) Start() {
	http.Handle("/push", p)

	p.syncNode.Run()
	select {}
}

func (p *Proxy) Push(k, v string) {
	println(p.addr, "push", k, v)
	msg := &Message{k, v}
	p.push(msg)

	// Sync: broadcast msg to peers
	p.syncNode.out <- msg
}

// store
func (p *Proxy) push(msg *Message) {
	println(p.addr, "push", msg.Key, msg.Val)
}

// proxy service handler
func (p *Proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	var msg = Message{}
	err = json.Unmarshal(data, &msg)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	p.Push(msg.Key, msg.Val)
}

// sync node def
type Node struct {
	id string

	// reader channel
	// read from remote idc as a consumer
	in     chan *Message
	reader Reader

	// broadcast channel is ready for broadcast
	// write to local as a producer
	out       chan *Message
	broadcast *Broadcast

	wg *sync.WaitGroup
}

type Reader interface {
	Recv() <-chan *Message
}

type Broadcast struct {
	peers   map[uint32]*Peer
	storeCh chan *Message
}

type Peer struct {
	ID   uint32
	Addr string
}

func newBroadcast(addrs []string) *Broadcast {
	broadcast := &Broadcast{
		peers:   make(map[uint32]*Peer),
		storeCh: make(chan *Message, 1024),
	}
	for i := 0; i < len(addrs); i++ {
		broadcast.peers[uint32(i)] = &Peer{
			Addr: addrs[i],
		}
	}
	go broadcast.broadcast()
	return broadcast
}

func (b *Broadcast) sync(msg *Message) {
	// local msg bus
	b.storeCh <- msg
}

func (b *Broadcast) broadcast() {
	for {
		select {
		case msg := <-b.storeCh:
			c := http.Client{}

			// should be replace by kafka_consumer_broadcast
			for _, p := range b.peers {
				bf := &bytes.Buffer{}
				json.NewEncoder(bf).Encode(msg)
				println("broadcast to", p.Addr, msg.Key, msg.Val)
				req, _ := http.NewRequest("POST", p.Addr, bf)
				c.Do(req)
			}
		}
	}
}

func (n *Node) Run() {
	go n.storeLoop()
	go n.recvLoop()
	go n.sendLoop()
}

func (n *Node) recvLoop() {
	n.wg.Add(1)
	defer n.wg.Done()

	in := n.reader.Recv()
	for {
		select {
		case msg := <-in:
			println(n.id, "recv", msg.Key, msg.Val)
			n.in <- msg
		}
	}
}

func (n *Node) storeLoop() {
	n.wg.Add(1)
	defer n.wg.Done()

	for {
		select {
		case msg := <-n.in:
			println(n.id, "push", msg.Key, msg.Val)
		}
	}
}

func (n *Node) sendLoop() {
	n.wg.Add(1)
	defer n.wg.Done()

	for {
		select {
		case msg := <-n.out:
			n.broadcast.sync(msg)
		}
	}
}

func (n *Node) Close() {
	n.wg.Wait()
}

type httpReader struct {
	addr string
	in   chan *Message
}

func NewHTTPReader(addr string) *httpReader {
	reader := &httpReader{
		addr: addr,
		in:   make(chan *Message, 1024),
	}
	go reader.serve()
	return reader
}

func (hr *httpReader) Recv() <-chan *Message {
	return hr.in
}

func (hr *httpReader) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	var msg = Message{}
	err = json.Unmarshal(data, &msg)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	hr.in <- &msg
}

func (hr *httpReader) serve() {
	http.Handle("/sync", hr)
	http.ListenAndServe(hr.addr, nil)
}

func main() {
	var (
		addr  string
		hosts string
	)
	flag.StringVar(&addr, "a", ":10002", "")
	flag.StringVar(&hosts, "h", "http://127.0.0.1:10003/sync;http://127.0.0.1:10004/sync", "")
	flag.Parse()
	proxy := NewProxy(addr, strings.Split(hosts, ";"))
	proxy.Start()
}
