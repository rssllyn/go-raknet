package extended

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"time"
	"unsafe"

	"go.uber.org/zap"

	"changit.cn/warrior_server/go-raknet"
	. "changit.cn/warrior_server/sercom/logger"
)

const (
	ConnPacketBuffer       = 100 // number of packets to keep and wait for handling
	acceptBacklog          = 128 // number of connections to keep and wait for accept
	shutDownNotifyDuration = 100 // milliseconds
)

type Conn struct {
	peer                raknet.RakPeerInterface
	buff                []byte // first packet data, buff it for reading multiple times
	buffReadIdx         int    // start index in buff that has not been read
	chData              chan []byte
	remoteAddressOrGUID raknet.AddressOrGUID

	// indicates whether the connection is opened as a client peer
	// the RakPeerInterface will only be ShutDown for client connection, which is expected to connect to only one server
	isClient bool

	done chan struct{}
}

// receiving packet for client peer
func (c *Conn) monitor() {
	for true {
		packet := c.peer.Receive()
		if unsafe.Pointer(packet.(raknet.SwigcptrPacket).Swigcptr()) == nil {
			// no packets received
			time.Sleep(10 * time.Millisecond)
			continue
		}
		c.handlePacket(packet)
	}
}

// hanldling packet for client peer
func (c *Conn) handlePacket(packet raknet.Packet) {
	defer c.peer.DeallocatePacket(packet)

	identifier := raknet.GetPacketIdentifier(packet)
	Logger.Info("packet received", zap.Int("message identifier", int(identifier)))
	switch raknet.DefaultMessageIDTypes(identifier) {
	case raknet.ID_USER_PACKET_ENUM:
		data := []byte(raknet.GetPacketPayload(packet))
		c.chData <- data
		Logger.Debug("packet data", zap.String("hex", hex.EncodeToString(data)))
	case raknet.ID_DISCONNECTION_NOTIFICATION:
		Logger.Debug("connection lost")
		close(c.done)
	case raknet.ID_CONNECTION_LOST:
		Logger.Debug("connection lost")
		close(c.done)
	}
}

func (c *Conn) Read(b []byte) (int, error) {
	for len(c.buff) == 0 {
		select {
		case c.buff = <-c.chData:
			c.buffReadIdx = 0
		case <-c.done:
			if len(c.chData) > 0 {
				Logger.Debug("connection closed, but still have data unprocessed, will process first")
			} else {
				return 0, io.EOF
			}
		}
	}
	n := copy(b, c.buff[c.buffReadIdx:])
	if n == len(c.buff)-c.buffReadIdx {
		c.buff = []byte{}
	} else {
		c.buffReadIdx += n
	}
	return n, nil
}

func (c *Conn) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}

	var buff bytes.Buffer
	buff.WriteByte(byte(raknet.ID_USER_PACKET_ENUM))
	buff.Write(b)

	// raknet requires the data sent (char* type in c++) to be ended with \0 character
	buff.WriteByte(0)

	sent := c.peer.Send(string(buff.Bytes()), buff.Len(), raknet.HIGH_PRIORITY, raknet.RELIABLE_ORDERED, byte(0), c.remoteAddressOrGUID, false)
	if int(sent.Swigcptr()) < buff.Len() {
		return int(sent.Swigcptr()), errors.New("not all data sent")
	}
	return len(b), nil
}

func (c *Conn) Close() error {
	if c.isClient {
		c.peer.Shutdown(uint(shutDownNotifyDuration))
		return nil
	} else {
		return nil
	}
}

func (c *Conn) LocalAddr() net.Addr {
	return nil
}

func (c *Conn) RemoteAddr() net.Addr {
	return nil
}

func (c *Conn) SetDeadline(t time.Time) error {
	return nil
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	return nil
}

// waitForConnectionAccepted blocks until a ID_CONNECTION_REQUEST_ACCEPTED packet is received from raknet, and returns true
// it returns false with an error if it determines that the connection could never be accepted
func waitForConnectionAccepted(peer raknet.RakPeerInterface) (raknet.AddressOrGUID, error) {
	for true {
		packet := peer.Receive()
		if unsafe.Pointer(packet.(raknet.SwigcptrPacket).Swigcptr()) == nil {
			// no packets received
			time.Sleep(10 * time.Millisecond)
			continue
		}
		identifier := raknet.GetPacketIdentifier(packet)
		switch raknet.DefaultMessageIDTypes(identifier) {
		case raknet.ID_CONNECTION_REQUEST_ACCEPTED:
			addressOrGUID := raknet.NewAddressOrGUID(packet)
			peer.DeallocatePacket(packet)
			return addressOrGUID, nil
		}
		peer.DeallocatePacket(packet)
	}
	return nil, nil
}

func Dial(raddr string) (net.Conn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", raddr)
	if err != nil {
		return nil, err
	}
	var ip string
	if udpAddr.IP != nil {
		ip = udpAddr.IP.String()
	}

	peer := raknet.RakPeerInterfaceGetInstance()
	socketDescriptor := raknet.NewSocketDescriptor(uint16(0), "")
	var maxConnectionCount uint = 1
	var socketDescriptorCount uint = 1
	peer.Startup(maxConnectionCount, socketDescriptor, socketDescriptorCount)
	var password string
	var passwordLength int
	peer.Connect(ip, uint16(udpAddr.Port), password, passwordLength)
	addressOrGUID, err := waitForConnectionAccepted(peer)
	if err != nil {
		return nil, err
	}
	conn := &Conn{
		chData:              make(chan []byte, ConnPacketBuffer),
		peer:                peer,
		remoteAddressOrGUID: addressOrGUID,
		isClient:            true,
		done:                make(chan struct{}),
	}
	go conn.monitor()
	return conn, nil
}

type Listener struct {
	peer      raknet.RakPeerInterface
	sessions  map[uint16]*Conn
	chAccepts chan *Conn
}

func (l *Listener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.chAccepts:
		return conn, nil
	}
}

func (l *Listener) Close() error {
	return nil
}

func (l *Listener) Addr() net.Addr {
	return nil
}

// monitor receiving packets from the connection, and create sessions if neccessary
func (l *Listener) monitor() {
	for true {
		packet := l.peer.Receive()
		if unsafe.Pointer(packet.(raknet.SwigcptrPacket).Swigcptr()) == nil {
			// no packets received
			time.Sleep(10 * time.Millisecond)
			continue
		}
		l.handlePacket(packet)
	}
}

func (l *Listener) handlePacket(packet raknet.Packet) {
	defer l.peer.DeallocatePacket(packet)

	identifier := raknet.GetPacketIdentifier(packet)
	Logger.Info("packet received", zap.Int("message identifier", int(identifier)))
	switch raknet.DefaultMessageIDTypes(identifier) {
	case raknet.ID_NEW_INCOMING_CONNECTION:
		if len(l.chAccepts) >= cap(l.chAccepts) {
			// prevent packet receiving of existing sessions being blocked
			return
		}
		sess := &Conn{
			chData:              make(chan []byte, ConnPacketBuffer),
			peer:                l.peer,
			remoteAddressOrGUID: raknet.NewAddressOrGUID(packet),
			isClient:            false,
			done:                make(chan struct{}),
		}

		// get unique ID for this session, notice that when raknet detects connection lost,
		// corresponding session ID would be reused in latter incomming connections
		sessionID := packet.GetGuid().GetSystemIndex()

		l.sessions[sessionID] = sess
		l.chAccepts <- sess
		Logger.Debug("new incoming connection received", zap.Uint16("session ID", sessionID))
	case raknet.ID_USER_PACKET_ENUM:
		data := []byte(raknet.GetPacketPayload(packet))
		sessionID := packet.GetGuid().GetSystemIndex()
		if sess, ok := l.sessions[sessionID]; ok {
			sess.chData <- data
			Logger.Debug("packet received for session", zap.Uint16("session ID", sessionID))
		}
	case raknet.ID_DISCONNECTION_NOTIFICATION:
		Logger.Debug("connection lost")
		sessionID := packet.GetGuid().GetSystemIndex()
		if sess, ok := l.sessions[sessionID]; ok {
			close(sess.done)
		}
	case raknet.ID_CONNECTION_LOST:
		Logger.Debug("connection lost")
		sessionID := packet.GetGuid().GetSystemIndex()
		if sess, ok := l.sessions[sessionID]; ok {
			close(sess.done)
		}
	}
}

func Listen(laddr string, maxConnections int) (net.Listener, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		return nil, err
	}
	var ip string
	if udpAddr.IP != nil {
		ip = udpAddr.IP.String()
	}

	peer := raknet.RakPeerInterfaceGetInstance()
	socketDescriptor := raknet.NewSocketDescriptor(uint16(udpAddr.Port), ip)
	var socketDescriptorCount uint = 1
	peer.Startup(uint(maxConnections), socketDescriptor, socketDescriptorCount)

	peer.SetMaximumIncomingConnections(uint16(maxConnections))

	l := &Listener{
		peer:      peer,
		sessions:  make(map[uint16]*Conn),
		chAccepts: make(chan *Conn, acceptBacklog),
	}

	go l.monitor()

	return l, nil
}
