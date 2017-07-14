%module raknet

%{
#include "RakPeerInterface.h"
#include "RakNetTypes.h"
#include "PacketPriority.h"
#include "MessageIdentifiers.h"
#include "RakNetTime.h"
%}

%inline %{
unsigned char GetPacketIdentifier(RakNet::Packet *p)
{
    if ((unsigned char)p->data[0] == ID_TIMESTAMP)
        return (unsigned char) p->data[sizeof(RakNet::MessageID) + sizeof(RakNet::Time)];
    else
        return (unsigned char) p->data[0];
}

char *GetPacketPayload(RakNet::Packet *p)
{
    if ((unsigned char)p->data[0] == ID_TIMESTAMP)
        return (char *)p->data + sizeof(RakNet::MessageID) + sizeof(RakNet::Time) + 1;
    else
        return (char *)p->data + 1;
}
%}

%include "RakNetTime.h"
%include "MessageIdentifiers.h"
%include "PacketPriority.h"

%rename(Copy) RakNet::SystemAddress::operator =;
%rename(Equals) RakNet::SystemAddress::operator ==;
%rename(NotEquals) RakNet::SystemAddress::operator !=;
%rename(GreaterThan) RakNet::SystemAddress::operator >;
%rename(LessThan) RakNet::SystemAddress::operator <;

namespace RakNet
{

typedef unsigned short SystemIndex;
struct Packet;
struct SystemAddress;

enum StartupResult
{
	RAKNET_STARTED,
	RAKNET_ALREADY_STARTED,
	INVALID_SOCKET_DESCRIPTORS,
	INVALID_MAX_CONNECTIONS,
	SOCKET_FAMILY_NOT_SUPPORTED,
	SOCKET_PORT_ALREADY_IN_USE,
	SOCKET_FAILED_TO_BIND,
	SOCKET_FAILED_TEST_SEND,
	PORT_CANNOT_BE_ZERO,
	FAILED_TO_CREATE_NETWORK_THREAD,
	COULD_NOT_GENERATE_GUID,
	STARTUP_OTHER_FAILURE
};
enum ConnectionAttemptResult
{
	CONNECTION_ATTEMPT_STARTED,
	INVALID_PARAMETER,
	CANNOT_RESOLVE_DOMAIN_NAME,
	ALREADY_CONNECTED_TO_ENDPOINT,
	CONNECTION_ATTEMPT_ALREADY_IN_PROGRESS,
	SECURITY_INITIALIZATION_FAILED
};

struct SocketDescriptor
{
	SocketDescriptor();
	SocketDescriptor(unsigned short _port, const char *_hostAddress);
	unsigned short port;
	char hostAddress[32];
	short socketFamily;
	unsigned short remotePortRakNetWasStartedOn_PS3_PSP2;
	int chromeInstance;
	bool blockingSocket;
	unsigned int extraSocketOptions;
};

struct RakNetGUID
{
	SystemIndex systemIndex;
};

struct AddressOrGUID
{
	AddressOrGUID( Packet *packet );
	RakNetGUID rakNetGuid;
	SystemAddress systemAddress;
};

struct SystemAddress
{
	/// Constructors
	SystemAddress();
	SystemAddress(const char *str);
	SystemAddress(const char *str, unsigned short port);

	/// This is not used internally, but holds a copy of the port held in the address union, so for debugging it's easier to check what port is being held
	unsigned short debugPort;

	/// \internal Return the size to write to a bitStream
	static int size(void);

	/// Hash the system address
	static unsigned long ToInteger( const SystemAddress &sa );

	/// Return the IP version, either IPV4 or IPV6
	/// \return Either 4 or 6
	unsigned char GetIPVersion(void) const;

	/// \internal Returns either IPPROTO_IP or IPPROTO_IPV6
	/// \sa GetIPVersion
	unsigned int GetIPPROTO(void) const;

	/// Call SetToLoopback(), with whatever IP version is currently held. Defaults to IPV4
	void SetToLoopback(void);

	/// Call SetToLoopback() with a specific IP version
	/// \param[in] ipVersion Either 4 for IPV4 or 6 for IPV6
	void SetToLoopback(unsigned char ipVersion);

	/// \return If was set to 127.0.0.1 or ::1
	bool IsLoopback(void) const;

	// Return the systemAddress as a string in the format <IP>|<Port>
	// Returns a static string
	// NOT THREADSAFE
	// portDelineator should not be '.', ':', '%', '-', '/', a number, or a-f
	const char *ToString(bool writePort=true, char portDelineator='|') const;

	// Return the systemAddress as a string in the format <IP>|<Port>
	// dest must be large enough to hold the output
	// portDelineator should not be '.', ':', '%', '-', '/', a number, or a-f
	// THREADSAFE
	void ToString(bool writePort, char *dest, char portDelineator='|') const;

	/// Set the system address from a printable IP string, for example "192.0.2.1" or "2001:db8:63b3:1::3490"
	/// You can write the port as well, using the portDelineator, for example "192.0.2.1|1234"
	/// \param[in] str A printable IP string, for example "192.0.2.1" or "2001:db8:63b3:1::3490". Pass 0 for \a str to set to UNASSIGNED_SYSTEM_ADDRESS
	/// \param[in] portDelineator if \a str contains a port, delineate the port with this character. portDelineator should not be '.', ':', '%', '-', '/', a number, or a-f
	/// \param[in] ipVersion Only used if str is a pre-defined address in the wrong format, such as 127.0.0.1 but you want ip version 6, so you can pass 6 here to do the conversion
	/// \note The current port is unchanged if a port is not specified in \a str
	/// \return True on success, false on ipVersion does not match type of passed string
	bool FromString(const char *str, char portDelineator='|', int ipVersion=0);

	/// Same as FromString(), but you explicitly set a port at the same time
	bool FromStringExplicitPort(const char *str, unsigned short port, int ipVersion=0);

	/// Copy the port from another SystemAddress structure
	void CopyPort( const SystemAddress& right );

	/// Returns if two system addresses have the same IP (port is not checked)
	bool EqualsExcludingPort( const SystemAddress& right ) const;

	/// Returns the port in host order (this is what you normally use)
	unsigned short GetPort(void) const;

	/// \internal Returns the port in network order
	unsigned short GetPortNetworkOrder(void) const;

	/// Sets the port. The port value should be in host order (this is what you normally use)
	/// Renamed from SetPort because of winspool.h http://edn.embarcadero.com/article/21494
	void SetPortHostOrder(unsigned short s);

	/// \internal Sets the port. The port value should already be in network order.
	void SetPortNetworkOrder(unsigned short s);

	/// Old version, for crap platforms that don't support newer socket functions
	bool SetBinaryAddress(const char *str, char portDelineator=':');
	/// Old version, for crap platforms that don't support newer socket functions
	void ToString_Old(bool writePort, char *dest, char portDelineator=':') const;

	/// \internal sockaddr_in6 requires extra data beyond just the IP and port. Copy that extra data from an existing SystemAddress that already has it
	void FixForIPVersion(const SystemAddress &boundAddressToSocket);

	bool IsLANAddress(void);

	SystemAddress& operator = ( const SystemAddress& input );
	bool operator==( const SystemAddress& right ) const;
	bool operator!=( const SystemAddress& right ) const;
	bool operator > ( const SystemAddress& right ) const;
	bool operator < ( const SystemAddress& right ) const;

	/// \internal Used internally for fast lookup. Optional (use -1 to do regular lookup). Don't transmit this.
	SystemIndex systemIndex;
};


struct Packet
{
	SystemAddress systemAddress;

	RakNetGUID guid;

	/// The length of the data in bytes
	unsigned int length;

	/// The length of the data in bits
	uint32_t bitSize;

	/// The data from the sender
	unsigned char* data;

	/// @internal
	/// Indicates whether to delete the data, or to simply delete the packet.
	bool deleteData;

	/// @internal
	/// If true, this message is meant for the user, not for the plugins, so do not process it through plugins
	bool wasGeneratedLocally;
};

enum PublicKeyMode
{
	/// The connection is insecure. You can also just pass 0 for the pointer to PublicKey in RakPeerInterface::Connect()
	PKM_INSECURE_CONNECTION,

	/// Accept whatever public key the server gives us. This is vulnerable to man in the middle, but does not require
	/// distribution of the public key in advance of connecting.
	PKM_ACCEPT_ANY_PUBLIC_KEY,

	/// Use a known remote server public key. PublicKey::remoteServerPublicKey must be non-zero.
	/// This is the recommended mode for secure connections.
	PKM_USE_KNOWN_PUBLIC_KEY,

	/// Use a known remote server public key AND provide a public key for the connecting client.
	/// PublicKey::remoteServerPublicKey, myPublicKey and myPrivateKey must be all be non-zero.
	/// The server must cooperate for this mode to work.
	/// I recommend not using this mode except for server-to-server communication as it significantly increases the CPU requirements during connections for both sides.
	/// Furthermore, when it is used, a connection password should be used as well to avoid DoS attacks.
	PKM_USE_TWO_WAY_AUTHENTICATION
};

/// Passed to RakPeerInterface::Connect()
struct PublicKey
{
	/// How to interpret the public key, see above
	PublicKeyMode publicKeyMode;

	/// Pointer to a public key of length cat::EasyHandshake::PUBLIC_KEY_BYTES. See the Encryption sample.
	char *remoteServerPublicKey;

	/// (Optional) Pointer to a public key of length cat::EasyHandshake::PUBLIC_KEY_BYTES
	char *myPublicKey;

	/// (Optional) Pointer to a private key of length cat::EasyHandshake::PRIVATE_KEY_BYTES
	char *myPrivateKey;
};

class RakPeerInterface
{
public:
	static RakPeerInterface *GetInstance(void);
        static void DestroyInstance(RakPeerInterface* x);
        RakPeerInterface();
	virtual ~RakPeerInterface()	{}

	virtual StartupResult Startup( unsigned int maxConnections, SocketDescriptor *socketDescriptors, unsigned socketDescriptorCount, int threadPriority=-99999 )=0;
	virtual void SetMaximumIncomingConnections( unsigned short numberAllowed )=0;

	virtual ConnectionAttemptResult Connect( const char* host, unsigned short remotePort, const char *passwordData, int passwordDataLength, PublicKey *publicKey=0, unsigned connectionSocketIndex=0, unsigned sendConnectionAttemptCount=12, unsigned timeBetweenSendConnectionAttemptsMS=500, RakNet::TimeMS timeoutTime=0 )=0;

	virtual uint32_t Send( const char *data, const int length, PacketPriority priority, PacketReliability reliability, char orderingChannel, const AddressOrGUID systemIdentifier, bool broadcast, uint32_t forceReceiptNumber=0 )=0;

	virtual Packet* Receive( void )=0;

	/// Call this to deallocate a message returned by Receive() when you are done handling it.
	/// \param[in] packet The message to deallocate.	
	virtual void DeallocatePacket( Packet *packet )=0;

	/// \brief Stops the network threads and closes all connections.
	/// \param[in] blockDuration How long, in milliseconds, you should wait for all remaining messages to go out, including ID_DISCONNECTION_NOTIFICATION.  If 0, it doesn't wait at all.
	/// \param[in] orderingChannel If blockDuration > 0, ID_DISCONNECTION_NOTIFICATION will be sent on this channel
	/// \param[in] disconnectionNotificationPriority Priority to send ID_DISCONNECTION_NOTIFICATION on.
	/// If you set it to 0 then the disconnection notification won't be sent
	virtual void Shutdown( unsigned int blockDuration, unsigned char orderingChannel=0, PacketPriority disconnectionNotificationPriority=LOW_PRIORITY )=0;
};

} // namespace RakNet
