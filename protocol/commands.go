package protocol

// Wire command numbers, named as in Luanti's networkprotocol.h.
const (
	TOCLIENT_HELLO                    uint16 = 0x02
	TOCLIENT_AUTH_ACCEPT              uint16 = 0x03
	TOCLIENT_ACCEPT_SUDO_MODE         uint16 = 0x04
	TOCLIENT_DENY_SUDO_MODE           uint16 = 0x05
	TOCLIENT_ACCESS_DENIED            uint16 = 0x0a
	TOCLIENT_BLOCKDATA                uint16 = 0x20
	TOCLIENT_ADDNODE                  uint16 = 0x21
	TOCLIENT_REMOVENODE               uint16 = 0x22
	TOCLIENT_INVENTORY                uint16 = 0x27
	TOCLIENT_TIME_OF_DAY              uint16 = 0x29
	TOCLIENT_CSM_RESTRICTION_FLAGS    uint16 = 0x2a
	TOCLIENT_PLAYER_SPEED             uint16 = 0x2b
	TOCLIENT_MEDIA_PUSH               uint16 = 0x2c
	TOCLIENT_CHAT_MESSAGE             uint16 = 0x2f
	TOCLIENT_ACTIVE_OBJECT_REMOVE_ADD uint16 = 0x31
	TOCLIENT_ACTIVE_OBJECT_MESSAGES   uint16 = 0x32
	TOCLIENT_HP                       uint16 = 0x33
	TOCLIENT_MOVE_PLAYER              uint16 = 0x34
	TOCLIENT_ACCESS_DENIED_LEGACY     uint16 = 0x35
	TOCLIENT_FOV                      uint16 = 0x36
	TOCLIENT_DEATHSCREEN_LEGACY       uint16 = 0x37
	TOCLIENT_MEDIA                    uint16 = 0x38
	TOCLIENT_NODEDEF                  uint16 = 0x3a
	TOCLIENT_ANNOUNCE_MEDIA           uint16 = 0x3c
	TOCLIENT_ITEMDEF                  uint16 = 0x3d
	TOCLIENT_PLAY_SOUND               uint16 = 0x3f
	TOCLIENT_STOP_SOUND               uint16 = 0x40
	TOCLIENT_PRIVILEGES               uint16 = 0x41
	TOCLIENT_INVENTORY_FORMSPEC       uint16 = 0x42
	TOCLIENT_DETACHED_INVENTORY       uint16 = 0x43
	TOCLIENT_SHOW_FORMSPEC            uint16 = 0x44
	TOCLIENT_MOVEMENT                 uint16 = 0x45
	TOCLIENT_SPAWN_PARTICLE           uint16 = 0x46
	TOCLIENT_ADD_PARTICLESPAWNER      uint16 = 0x47
	TOCLIENT_CAMERA                   uint16 = 0x48
	TOCLIENT_HUDADD                   uint16 = 0x49
	TOCLIENT_HUDRM                    uint16 = 0x4a
	TOCLIENT_HUDCHANGE                uint16 = 0x4b
	TOCLIENT_HUD_SET_FLAGS            uint16 = 0x4c
	TOCLIENT_HUD_SET_PARAM            uint16 = 0x4d
	TOCLIENT_BREATH                   uint16 = 0x4e
	TOCLIENT_SET_SKY                  uint16 = 0x4f
	TOCLIENT_OVERRIDE_DAY_NIGHT_RATIO uint16 = 0x50
	TOCLIENT_LOCAL_PLAYER_ANIMATIONS  uint16 = 0x51
	TOCLIENT_EYE_OFFSET               uint16 = 0x52
	TOCLIENT_DELETE_PARTICLESPAWNER   uint16 = 0x53
	TOCLIENT_CLOUD_PARAMS             uint16 = 0x54
	TOCLIENT_FADE_SOUND               uint16 = 0x55
	TOCLIENT_UPDATE_PLAYER_LIST       uint16 = 0x56
	TOCLIENT_MODCHANNEL_MSG           uint16 = 0x57
	TOCLIENT_MODCHANNEL_SIGNAL        uint16 = 0x58
	TOCLIENT_NODEMETA_CHANGED         uint16 = 0x59
	TOCLIENT_SET_SUN                  uint16 = 0x5a
	TOCLIENT_SET_MOON                 uint16 = 0x5b
	TOCLIENT_SET_STARS                uint16 = 0x5c
	TOCLIENT_MOVE_PLAYER_REL          uint16 = 0x5d
	TOCLIENT_SRP_BYTES_S_B            uint16 = 0x60
	TOCLIENT_FORMSPEC_PREPEND         uint16 = 0x61
	TOCLIENT_MINIMAP_MODES            uint16 = 0x62
	TOCLIENT_SET_LIGHTING             uint16 = 0x63
	TOCLIENT_SPAWN_PARTICLE_BATCH     uint16 = 0x64

	TOSERVER_INIT               uint16 = 0x02
	TOSERVER_INIT2              uint16 = 0x11
	TOSERVER_MODCHANNEL_JOIN    uint16 = 0x17
	TOSERVER_MODCHANNEL_LEAVE   uint16 = 0x18
	TOSERVER_MODCHANNEL_MSG     uint16 = 0x19
	TOSERVER_PLAYERPOS          uint16 = 0x23
	TOSERVER_GOTBLOCKS          uint16 = 0x24
	TOSERVER_DELETEDBLOCKS      uint16 = 0x25
	TOSERVER_INVENTORY_ACTION   uint16 = 0x31
	TOSERVER_CHAT_MESSAGE       uint16 = 0x32
	TOSERVER_DAMAGE             uint16 = 0x35
	TOSERVER_PLAYERITEM         uint16 = 0x37
	TOSERVER_RESPAWN_LEGACY     uint16 = 0x38
	TOSERVER_INTERACT           uint16 = 0x39
	TOSERVER_REMOVED_SOUNDS     uint16 = 0x3a
	TOSERVER_NODEMETA_FIELDS    uint16 = 0x3b
	TOSERVER_INVENTORY_FIELDS   uint16 = 0x3c
	TOSERVER_REQUEST_MEDIA      uint16 = 0x40
	TOSERVER_HAVE_MEDIA         uint16 = 0x41
	TOSERVER_CLIENT_READY       uint16 = 0x43
	TOSERVER_FIRST_SRP          uint16 = 0x50
	TOSERVER_SRP_BYTES_A        uint16 = 0x51
	TOSERVER_SRP_BYTES_M        uint16 = 0x52
	TOSERVER_UPDATE_CLIENT_INFO uint16 = 0x53
)

// Transport describes how Luanti sends a client packet: on which channel and
// whether reliably.
type Transport struct {
	Channel  uint8
	Reliable bool
}

// ToServerTransport returns the channel and reliability Luanti clients use for a
// client-to-server command.
func ToServerTransport(cmd uint16) Transport {
	switch cmd {
	case TOSERVER_INIT:
		return Transport{1, false}
	case TOSERVER_INIT2:
		return Transport{1, true}
	case TOSERVER_MODCHANNEL_JOIN:
		return Transport{0, true}
	case TOSERVER_MODCHANNEL_LEAVE:
		return Transport{0, true}
	case TOSERVER_MODCHANNEL_MSG:
		return Transport{0, true}
	case TOSERVER_PLAYERPOS:
		return Transport{0, false}
	case TOSERVER_GOTBLOCKS:
		return Transport{2, true}
	case TOSERVER_DELETEDBLOCKS:
		return Transport{2, true}
	case TOSERVER_INVENTORY_ACTION:
		return Transport{0, true}
	case TOSERVER_CHAT_MESSAGE:
		return Transport{0, true}
	case TOSERVER_DAMAGE:
		return Transport{0, true}
	case TOSERVER_PLAYERITEM:
		return Transport{0, true}
	case TOSERVER_RESPAWN_LEGACY:
		return Transport{0, true}
	case TOSERVER_INTERACT:
		return Transport{0, true}
	case TOSERVER_REMOVED_SOUNDS:
		return Transport{2, true}
	case TOSERVER_NODEMETA_FIELDS:
		return Transport{0, true}
	case TOSERVER_INVENTORY_FIELDS:
		return Transport{0, true}
	case TOSERVER_REQUEST_MEDIA:
		return Transport{1, true}
	case TOSERVER_HAVE_MEDIA:
		return Transport{2, true}
	case TOSERVER_CLIENT_READY:
		return Transport{1, true}
	case TOSERVER_FIRST_SRP:
		return Transport{1, true}
	case TOSERVER_SRP_BYTES_A:
		return Transport{1, true}
	case TOSERVER_SRP_BYTES_M:
		return Transport{1, true}
	case TOSERVER_UPDATE_CLIENT_INFO:
		return Transport{2, true}
	}
	return Transport{0, true}
}
