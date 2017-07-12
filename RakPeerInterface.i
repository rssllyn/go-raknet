%module raknet

%{
#include "RakPeerInterface.h"
#include "RakNetTypes.h"
#include "PacketPriority.h"
%}

%include "PacketPriority.h"

namespace RakNet
{
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
	RakNetGUID();
	explicit RakNetGUID(uint64_t _g) {g=_g; systemIndex=(SystemIndex)-1;}
	uint64_t g;

	// Return the GUID as a string
	// Returns a static string
	// NOT THREADSAFE
	const char *ToString(void) const;

	// Return the GUID as a string
	// dest must be large enough to hold the output
	// THREADSAFE
	void ToString(char *dest) const;

	bool FromString(const char *source);

	static unsigned long ToUint32( const RakNetGUID &g );

	RakNetGUID& operator = ( const RakNetGUID& input )
	{
		g=input.g;
		systemIndex=input.systemIndex;
		return *this;
	}

	// Used internally for fast lookup. Optional (use -1 to do regular lookup). Don't transmit this.
	SystemIndex systemIndex;
	static int size() {return (int) sizeof(uint64_t);}

	bool operator==( const RakNetGUID& right ) const;
	bool operator!=( const RakNetGUID& right ) const;
	bool operator > ( const RakNetGUID& right ) const;
	bool operator < ( const RakNetGUID& right ) const;
};

struct RAK_DLL_EXPORT AddressOrGUID
{
	RakNetGUID rakNetGuid;
	SystemAddress systemAddress;

	SystemIndex GetSystemIndex(void) const {if (rakNetGuid!=UNASSIGNED_RAKNET_GUID) return rakNetGuid.systemIndex; else return systemAddress.systemIndex;}
	bool IsUndefined(void) const {return rakNetGuid==UNASSIGNED_RAKNET_GUID && systemAddress==UNASSIGNED_SYSTEM_ADDRESS;}
	void SetUndefined(void) {rakNetGuid=UNASSIGNED_RAKNET_GUID; systemAddress=UNASSIGNED_SYSTEM_ADDRESS;}
	static unsigned long ToInteger( const AddressOrGUID &aog );
	const char *ToString(bool writePort=true) const;
	void ToString(bool writePort, char *dest) const;

	AddressOrGUID() {}
	AddressOrGUID( const AddressOrGUID& input )
	{
		rakNetGuid=input.rakNetGuid;
		systemAddress=input.systemAddress;
	}
	AddressOrGUID( const SystemAddress& input )
	{
		rakNetGuid=UNASSIGNED_RAKNET_GUID;
		systemAddress=input;
	}
	AddressOrGUID( Packet *packet );
	AddressOrGUID( const RakNetGUID& input )
	{
		rakNetGuid=input;
		systemAddress=UNASSIGNED_SYSTEM_ADDRESS;
	}
	AddressOrGUID& operator = ( const AddressOrGUID& input )
	{
		rakNetGuid=input.rakNetGuid;
		systemAddress=input.systemAddress;
		return *this;
	}

	AddressOrGUID& operator = ( const SystemAddress& input )
	{
		rakNetGuid=UNASSIGNED_RAKNET_GUID;
		systemAddress=input;
		return *this;
	}

	AddressOrGUID& operator = ( const RakNetGUID& input )
	{
		rakNetGuid=input;
		systemAddress=UNASSIGNED_SYSTEM_ADDRESS;
		return *this;
	}

	inline bool operator==( const AddressOrGUID& right ) const {return (rakNetGuid!=UNASSIGNED_RAKNET_GUID && rakNetGuid==right.rakNetGuid) || (systemAddress!=UNASSIGNED_SYSTEM_ADDRESS && systemAddress==right.systemAddress);}
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
	virtual uint32_t Send( const char *data, const int length, PacketPriority priority, PacketReliability reliability, char orderingChannel, const AddressOrGUID systemIdentifier, bool broadcast, uint32_t forceReceiptNumber=0 )=0;
};

} // namespace RakNet

