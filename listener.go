package raknet

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"time"
	"unsafe"

	"go.uber.org/zap"

	"github.com/rssllyn/go-raknet/wrapper"
)

const (
	ConnPacketBuffer       = 100 // number of packets to keep and wait for handling
	acceptBacklog          = 128 // number of connections to keep and wait for accept
	shutDownNotifyDuration = 100 // milliseconds
)

var Logger *zap.Logger

func init() {
	Logger, _ = zap.NewDevelopment()
}

type Conn struct {
	peer                wrapper.RakPeerInterface
	buff                []byte                // first packet data, buff it for reading multiple times
	buffReadIdx         int                   // start index in buff that has not been read
	chData              chan []byte           //
	remoteAddressOrGUID wrapper.AddressOrGUID // used to specify remote peer when Send
	localAddress        net.Addr
	remoteAddress       net.Addr

	// indicates whether the connection is opened as a client peer
	// the RakPeerInterface will only be ShutDown for client connection, which is expected to connect to only one server
	isClient bool

	done chan struct{}
}

// receiving packet for client peer
func (c *Conn) monitor() {
	for true {
		packet := c.peer.Receive()
		if unsafe.Pointer(packet.(wrapper.SwigcptrPacket).Swigcptr()) == nil {
			// no packets received
			time.Sleep(10 * time.Millisecond)
			continue
		}
		c.handlePacket(packet)
	}
}

// hanldling packet for client peer
func (c *Conn) handlePacket(packet wrapper.Packet) {
	defer c.peer.DeallocatePacket(packet)

	identifier := wrapper.GetPacketIdentifier(packet)
	Logger.Info("packet received", zap.Int("message identifier", int(identifier)))
	switch wrapper.DefaultMessageIDTypes(identifier) {
	case wrapper.ID_USER_PACKET_ENUM:
		data := []byte(wrapper.GetPacketPayload(packet))
		c.chData <- data
		Logger.Debug("packet data", zap.String("hex", hex.EncodeToString(data)))
	case wrapper.ID_DISCONNECTION_NOTIFICATION:
		Logger.Debug("connection lost")
		close(c.done)
	case wrapper.ID_CONNECTION_LOST:
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
	buff.WriteByte(byte(wrapper.ID_USER_PACKET_ENUM))
	buff.Write(b)

	// wrapper requires the data sent (char* type in c++) to be ended with \0 character
	buff.WriteByte(0)

	sent := c.peer.Send(string(buff.Bytes()), buff.Len(), wrapper.HIGH_PRIORITY, wrapper.RELIABLE_ORDERED, byte(0), c.remoteAddressOrGUID, false)
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
	return c.localAddress
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.remoteAddress
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

// waitForConnectionAccepted blocks until a ID_CONNECTION_REQUEST_ACCEPTED packet is received from wrapper, and returns true
// it returns false with an error if it determines that the connection could never be accepted
func waitForConnectionAccepted(peer wrapper.RakPeerInterface) (wrapper.AddressOrGUID, error) {
	for true {
		packet := peer.Receive()
		if unsafe.Pointer(packet.(wrapper.SwigcptrPacket).Swigcptr()) == nil {
			// no packets received
			time.Sleep(10 * time.Millisecond)
			continue
		}
		identifier := wrapper.GetPacketIdentifier(packet)
		switch wrapper.DefaultMessageIDTypes(identifier) {
		case wrapper.ID_CONNECTION_REQUEST_ACCEPTED:
			addressOrGUID := wrapper.NewAddressOrGUID(packet)
			peer.DeallocatePacket(packet)
			return addressOrGUID, nil
		}
		peer.DeallocatePacket(packet)
	}
	return nil, nil
}

func Dial(raddr string) (net.Conn, error) {
	remoteUDPAddr, err := net.ResolveUDPAddr("udp", raddr)
	if err != nil {
		return nil, err
	}
	if remoteUDPAddr.IP == nil {
		return nil, errors.New("no server ip address specified")
	}
	Logger.Debug("connecting to server", zap.String("server address", remoteUDPAddr.String()))

	peer := wrapper.RakPeerInterfaceGetInstance()
	socketDescriptor := wrapper.NewSocketDescriptor(uint16(0), "")
	Logger.Debug(
		"local address before connect",
		zap.String("host", socketDescriptor.GetHostAddress()),
		zap.Uint16("port", socketDescriptor.GetPort()),
	)
	var maxConnectionCount uint = 1
	var socketDescriptorCount uint = 1
	peer.Startup(maxConnectionCount, socketDescriptor, socketDescriptorCount)
	var password string
	var passwordLength int
	peer.Connect(remoteUDPAddr.IP.String(), uint16(remoteUDPAddr.Port), password, passwordLength)
	addressOrGUID, err := waitForConnectionAccepted(peer)
	if err != nil {
		return nil, err
	}
	Logger.Debug(
		"local address after connected",
		zap.String("host", socketDescriptor.GetHostAddress()),
		zap.Uint16("port", socketDescriptor.GetPort()),
	)

	conn := &Conn{
		chData:              make(chan []byte, ConnPacketBuffer),
		peer:                peer,
		remoteAddressOrGUID: addressOrGUID,
		isClient:            true,
		done:                make(chan struct{}),
		remoteAddress:       remoteUDPAddr,
	}
	go conn.monitor()
	return conn, nil
}

type Listener struct {
	peer          wrapper.RakPeerInterface
	sessions      map[uint16]*Conn
	chAccepts     chan *Conn
	listenAddress net.Addr
}

func (l *Listener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.chAccepts:
		return conn, nil
	}
}

func (l *Listener) Close() error {
	l.peer.Shutdown(uint(shutDownNotifyDuration))
	return nil
}

func (l *Listener) Addr() net.Addr {
	return l.listenAddress
}

// monitor receiving packets from the connection, and create sessions if neccessary
func (l *Listener) monitor() {
	for true {
		packet := l.peer.Receive()
		if unsafe.Pointer(packet.(wrapper.SwigcptrPacket).Swigcptr()) == nil {
			// no packets received
			time.Sleep(10 * time.Millisecond)
			continue
		}
		l.handlePacket(packet)
	}
}

func (l *Listener) handlePacket(packet wrapper.Packet) {
	defer l.peer.DeallocatePacket(packet)

	identifier := wrapper.GetPacketIdentifier(packet)
	Logger.Info("packet received", zap.Int("message identifier", int(identifier)))
	switch wrapper.DefaultMessageIDTypes(identifier) {
	case wrapper.ID_NEW_INCOMING_CONNECTION:
		if len(l.chAccepts) >= cap(l.chAccepts) {
			// prevent packet receiving of existing sessions being blocked
			return
		}
		ra := packet.GetSystemAddress().ToString(true, []byte(":")[0]).(string)
		Logger.Debug("remote address", zap.String("ip:port", ra))
		remoteAddress, err := net.ResolveUDPAddr("udp", ra)
		if err != nil {
			Logger.Warn("failed to get remote address", zap.Error(err))
		}
		sess := &Conn{
			chData:              make(chan []byte, ConnPacketBuffer),
			peer:                l.peer,
			remoteAddressOrGUID: wrapper.NewAddressOrGUID(packet),
			isClient:            false,
			done:                make(chan struct{}),
			localAddress:        l.listenAddress,
			remoteAddress:       remoteAddress,
		}

		// get unique ID for this session, notice that when wrapper detects connection lost,
		// corresponding session ID would be reused in latter incomming connections
		sessionID := packet.GetGuid().GetSystemIndex()

		l.sessions[sessionID] = sess
		l.chAccepts <- sess
		Logger.Debug("new incoming connection received", zap.Uint16("session ID", sessionID))
	case wrapper.ID_USER_PACKET_ENUM:
		data := []byte(wrapper.GetPacketPayload(packet))
		sessionID := packet.GetGuid().GetSystemIndex()
		if sess, ok := l.sessions[sessionID]; ok {
			sess.chData <- data
			Logger.Debug("packet received for session", zap.Uint16("session ID", sessionID))
		}
	case wrapper.ID_DISCONNECTION_NOTIFICATION:
		Logger.Debug("connection lost")
		sessionID := packet.GetGuid().GetSystemIndex()
		if sess, ok := l.sessions[sessionID]; ok {
			close(sess.done)
		}
	case wrapper.ID_CONNECTION_LOST:
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

	peer := wrapper.RakPeerInterfaceGetInstance()
	socketDescriptor := wrapper.NewSocketDescriptor(uint16(udpAddr.Port), ip)
	var socketDescriptorCount uint = 1
	peer.Startup(uint(maxConnections), socketDescriptor, socketDescriptorCount)

	peer.SetMaximumIncomingConnections(uint16(maxConnections))

	l := &Listener{
		peer:          peer,
		sessions:      make(map[uint16]*Conn),
		chAccepts:     make(chan *Conn, acceptBacklog),
		listenAddress: udpAddr,
	}

	go l.monitor()

	return l, nil
}
