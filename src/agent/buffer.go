package agent

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync/atomic"
)

import (
	"misc/packet"
)

const (
	BUFSIZE = 65535
	MAXCHAN = 4096
)

type _RawPacket struct {
	size uint16 // payload size
	data []byte // payload
}

type Buffer struct {
	ctrl    chan int
	pending chan *_RawPacket // pending Packet
	size    int32            // packet payload bytes count

	conn net.Conn // connection
}

//--------------------------------------------------------- Send packet
func (buf *Buffer) Send(data []byte) (err error) {
	if buf.size <= BUFSIZE {
		rp := _RawPacket{size: uint16(len(data)), data: data}
		buf.pending <- &rp

		atomic.AddInt32(&buf.size, int32(len(data)))
		return nil
	}

	return errors.New(fmt.Sprintf("Send Buffer Overflow, send rejected, possible DoS attack. Remote: %v", buf.conn.RemoteAddr()))
}

//--------------------------------------------------------- packet sender goroutine
func (buf *Buffer) Start() {
	defer recover()

	for {
		select {
		case pkt := <-buf.pending:
			buf.raw_send(pkt)
			atomic.AddInt32(&buf.size, -int32(len(pkt.data)))
		case _ = <-buf.ctrl:
			return
		}
	}
}

func (buf *Buffer) raw_send(pkt *_RawPacket) {
	headwriter := packet.Writer()
	headwriter.WriteU16(uint16(len(pkt.data)))

	_, err := buf.conn.Write(headwriter.Data())
	if err != nil {
		log.Println("Error send reply header:", err.Error())
		return
	}

	_, err = buf.conn.Write(pkt.data)
	if err != nil {
		log.Println("Error send reply msg:", err.Error())
		return
	}

	return
}

func NewBuffer(conn net.Conn, ctrl chan int) *Buffer {
	buf := Buffer{conn: conn, size: 0}
	buf.pending = make(chan *_RawPacket, MAXCHAN)
	buf.ctrl = ctrl
	return &buf
}