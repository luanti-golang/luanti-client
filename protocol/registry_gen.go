// Code generated from the command tables; DO NOT EDIT.

package protocol

func (*ToClientHello) Command() uint16                      { return TOCLIENT_HELLO }
func (*ToClientHello) Direction() Direction                 { return ToClient }
func (*ToClientAuthAccept) Command() uint16                 { return TOCLIENT_AUTH_ACCEPT }
func (*ToClientAuthAccept) Direction() Direction            { return ToClient }
func (*ToClientAcceptSudoMode) Command() uint16             { return TOCLIENT_ACCEPT_SUDO_MODE }
func (*ToClientAcceptSudoMode) Direction() Direction        { return ToClient }
func (*ToClientDenySudoMode) Command() uint16               { return TOCLIENT_DENY_SUDO_MODE }
func (*ToClientDenySudoMode) Direction() Direction          { return ToClient }
func (*ToClientAccessDenied) Command() uint16               { return TOCLIENT_ACCESS_DENIED }
func (*ToClientAccessDenied) Direction() Direction          { return ToClient }
func (*ToClientBlockData) Command() uint16                  { return TOCLIENT_BLOCKDATA }
func (*ToClientBlockData) Direction() Direction             { return ToClient }
func (*ToClientAddNode) Command() uint16                    { return TOCLIENT_ADDNODE }
func (*ToClientAddNode) Direction() Direction               { return ToClient }
func (*ToClientRemoveNode) Command() uint16                 { return TOCLIENT_REMOVENODE }
func (*ToClientRemoveNode) Direction() Direction            { return ToClient }
func (*ToClientInventory) Command() uint16                  { return TOCLIENT_INVENTORY }
func (*ToClientInventory) Direction() Direction             { return ToClient }
func (*ToClientTimeOfDay) Command() uint16                  { return TOCLIENT_TIME_OF_DAY }
func (*ToClientTimeOfDay) Direction() Direction             { return ToClient }
func (*ToClientCSMRestrictionFlags) Command() uint16        { return TOCLIENT_CSM_RESTRICTION_FLAGS }
func (*ToClientCSMRestrictionFlags) Direction() Direction   { return ToClient }
func (*ToClientPlayerSpeed) Command() uint16                { return TOCLIENT_PLAYER_SPEED }
func (*ToClientPlayerSpeed) Direction() Direction           { return ToClient }
func (*ToClientMediaPush) Command() uint16                  { return TOCLIENT_MEDIA_PUSH }
func (*ToClientMediaPush) Direction() Direction             { return ToClient }
func (*ToClientChatMessage) Command() uint16                { return TOCLIENT_CHAT_MESSAGE }
func (*ToClientChatMessage) Direction() Direction           { return ToClient }
func (*ToClientActiveObjectRemoveAdd) Command() uint16      { return TOCLIENT_ACTIVE_OBJECT_REMOVE_ADD }
func (*ToClientActiveObjectRemoveAdd) Direction() Direction { return ToClient }
func (*ToClientActiveObjectMessages) Command() uint16       { return TOCLIENT_ACTIVE_OBJECT_MESSAGES }
func (*ToClientActiveObjectMessages) Direction() Direction  { return ToClient }
func (*ToClientHP) Command() uint16                         { return TOCLIENT_HP }
func (*ToClientHP) Direction() Direction                    { return ToClient }
func (*ToClientMovePlayer) Command() uint16                 { return TOCLIENT_MOVE_PLAYER }
func (*ToClientMovePlayer) Direction() Direction            { return ToClient }
func (*ToClientAccessDeniedLegacy) Command() uint16         { return TOCLIENT_ACCESS_DENIED_LEGACY }
func (*ToClientAccessDeniedLegacy) Direction() Direction    { return ToClient }
func (*ToClientFov) Command() uint16                        { return TOCLIENT_FOV }
func (*ToClientFov) Direction() Direction                   { return ToClient }
func (*ToClientDeathscreenLegacy) Command() uint16          { return TOCLIENT_DEATHSCREEN_LEGACY }
func (*ToClientDeathscreenLegacy) Direction() Direction     { return ToClient }
func (*ToClientMedia) Command() uint16                      { return TOCLIENT_MEDIA }
func (*ToClientMedia) Direction() Direction                 { return ToClient }
func (*ToClientNodeDef) Command() uint16                    { return TOCLIENT_NODEDEF }
func (*ToClientNodeDef) Direction() Direction               { return ToClient }
func (*ToClientAnnounceMedia) Command() uint16              { return TOCLIENT_ANNOUNCE_MEDIA }
func (*ToClientAnnounceMedia) Direction() Direction         { return ToClient }
func (*ToClientItemDef) Command() uint16                    { return TOCLIENT_ITEMDEF }
func (*ToClientItemDef) Direction() Direction               { return ToClient }
func (*ToClientPlaySound) Command() uint16                  { return TOCLIENT_PLAY_SOUND }
func (*ToClientPlaySound) Direction() Direction             { return ToClient }
func (*ToClientStopSound) Command() uint16                  { return TOCLIENT_STOP_SOUND }
func (*ToClientStopSound) Direction() Direction             { return ToClient }
func (*ToClientPrivileges) Command() uint16                 { return TOCLIENT_PRIVILEGES }
func (*ToClientPrivileges) Direction() Direction            { return ToClient }
func (*ToClientInventoryFormspec) Command() uint16          { return TOCLIENT_INVENTORY_FORMSPEC }
func (*ToClientInventoryFormspec) Direction() Direction     { return ToClient }
func (*ToClientDetachedInventory) Command() uint16          { return TOCLIENT_DETACHED_INVENTORY }
func (*ToClientDetachedInventory) Direction() Direction     { return ToClient }
func (*ToClientShowFormspec) Command() uint16               { return TOCLIENT_SHOW_FORMSPEC }
func (*ToClientShowFormspec) Direction() Direction          { return ToClient }
func (*ToClientMovement) Command() uint16                   { return TOCLIENT_MOVEMENT }
func (*ToClientMovement) Direction() Direction              { return ToClient }
func (*ToClientSpawnParticle) Command() uint16              { return TOCLIENT_SPAWN_PARTICLE }
func (*ToClientSpawnParticle) Direction() Direction         { return ToClient }
func (*ToClientAddParticleSpawner) Command() uint16         { return TOCLIENT_ADD_PARTICLESPAWNER }
func (*ToClientAddParticleSpawner) Direction() Direction    { return ToClient }
func (*ToClientSpawnParticleBatch) Command() uint16         { return TOCLIENT_SPAWN_PARTICLE_BATCH }
func (*ToClientSpawnParticleBatch) Direction() Direction    { return ToClient }
func (*ToClientCamera) Command() uint16                     { return TOCLIENT_CAMERA }
func (*ToClientCamera) Direction() Direction                { return ToClient }
func (*ToClientHudAdd) Command() uint16                     { return TOCLIENT_HUDADD }
func (*ToClientHudAdd) Direction() Direction                { return ToClient }
func (*ToClientHudRm) Command() uint16                      { return TOCLIENT_HUDRM }
func (*ToClientHudRm) Direction() Direction                 { return ToClient }
func (*ToClientHudChange) Command() uint16                  { return TOCLIENT_HUDCHANGE }
func (*ToClientHudChange) Direction() Direction             { return ToClient }
func (*ToClientHudSetFlags) Command() uint16                { return TOCLIENT_HUD_SET_FLAGS }
func (*ToClientHudSetFlags) Direction() Direction           { return ToClient }
func (*ToClientHudSetParam) Command() uint16                { return TOCLIENT_HUD_SET_PARAM }
func (*ToClientHudSetParam) Direction() Direction           { return ToClient }
func (*ToClientBreath) Command() uint16                     { return TOCLIENT_BREATH }
func (*ToClientBreath) Direction() Direction                { return ToClient }
func (*ToClientSetSky) Command() uint16                     { return TOCLIENT_SET_SKY }
func (*ToClientSetSky) Direction() Direction                { return ToClient }
func (*ToClientOverrideDayNightRatio) Command() uint16      { return TOCLIENT_OVERRIDE_DAY_NIGHT_RATIO }
func (*ToClientOverrideDayNightRatio) Direction() Direction { return ToClient }
func (*ToClientLocalPlayerAnimations) Command() uint16      { return TOCLIENT_LOCAL_PLAYER_ANIMATIONS }
func (*ToClientLocalPlayerAnimations) Direction() Direction { return ToClient }
func (*ToClientEyeOffset) Command() uint16                  { return TOCLIENT_EYE_OFFSET }
func (*ToClientEyeOffset) Direction() Direction             { return ToClient }
func (*ToClientDeleteParticleSpawner) Command() uint16      { return TOCLIENT_DELETE_PARTICLESPAWNER }
func (*ToClientDeleteParticleSpawner) Direction() Direction { return ToClient }
func (*ToClientCloudParams) Command() uint16                { return TOCLIENT_CLOUD_PARAMS }
func (*ToClientCloudParams) Direction() Direction           { return ToClient }
func (*ToClientFadeSound) Command() uint16                  { return TOCLIENT_FADE_SOUND }
func (*ToClientFadeSound) Direction() Direction             { return ToClient }
func (*ToClientUpdatePlayerList) Command() uint16           { return TOCLIENT_UPDATE_PLAYER_LIST }
func (*ToClientUpdatePlayerList) Direction() Direction      { return ToClient }
func (*ToClientModChannelMsg) Command() uint16              { return TOCLIENT_MODCHANNEL_MSG }
func (*ToClientModChannelMsg) Direction() Direction         { return ToClient }
func (*ToClientModChannelSignal) Command() uint16           { return TOCLIENT_MODCHANNEL_SIGNAL }
func (*ToClientModChannelSignal) Direction() Direction      { return ToClient }
func (*ToClientNodemetaChanged) Command() uint16            { return TOCLIENT_NODEMETA_CHANGED }
func (*ToClientNodemetaChanged) Direction() Direction       { return ToClient }
func (*ToClientSetSun) Command() uint16                     { return TOCLIENT_SET_SUN }
func (*ToClientSetSun) Direction() Direction                { return ToClient }
func (*ToClientSetMoon) Command() uint16                    { return TOCLIENT_SET_MOON }
func (*ToClientSetMoon) Direction() Direction               { return ToClient }
func (*ToClientSetStars) Command() uint16                   { return TOCLIENT_SET_STARS }
func (*ToClientSetStars) Direction() Direction              { return ToClient }
func (*ToClientMovePlayerRel) Command() uint16              { return TOCLIENT_MOVE_PLAYER_REL }
func (*ToClientMovePlayerRel) Direction() Direction         { return ToClient }
func (*ToClientSRPBytesSB) Command() uint16                 { return TOCLIENT_SRP_BYTES_S_B }
func (*ToClientSRPBytesSB) Direction() Direction            { return ToClient }
func (*ToClientFormspecPrepend) Command() uint16            { return TOCLIENT_FORMSPEC_PREPEND }
func (*ToClientFormspecPrepend) Direction() Direction       { return ToClient }
func (*ToClientMinimapModes) Command() uint16               { return TOCLIENT_MINIMAP_MODES }
func (*ToClientMinimapModes) Direction() Direction          { return ToClient }
func (*ToClientSetLighting) Command() uint16                { return TOCLIENT_SET_LIGHTING }
func (*ToClientSetLighting) Direction() Direction           { return ToClient }
func (*ToServerInit) Command() uint16                       { return TOSERVER_INIT }
func (*ToServerInit) Direction() Direction                  { return ToServer }
func (*ToServerInit2) Command() uint16                      { return TOSERVER_INIT2 }
func (*ToServerInit2) Direction() Direction                 { return ToServer }
func (*ToServerModChannelJoin) Command() uint16             { return TOSERVER_MODCHANNEL_JOIN }
func (*ToServerModChannelJoin) Direction() Direction        { return ToServer }
func (*ToServerModChannelLeave) Command() uint16            { return TOSERVER_MODCHANNEL_LEAVE }
func (*ToServerModChannelLeave) Direction() Direction       { return ToServer }
func (*ToServerModChannelMsg) Command() uint16              { return TOSERVER_MODCHANNEL_MSG }
func (*ToServerModChannelMsg) Direction() Direction         { return ToServer }
func (*ToServerPlayerPos) Command() uint16                  { return TOSERVER_PLAYERPOS }
func (*ToServerPlayerPos) Direction() Direction             { return ToServer }
func (*ToServerGotBlocks) Command() uint16                  { return TOSERVER_GOTBLOCKS }
func (*ToServerGotBlocks) Direction() Direction             { return ToServer }
func (*ToServerDeletedBlocks) Command() uint16              { return TOSERVER_DELETEDBLOCKS }
func (*ToServerDeletedBlocks) Direction() Direction         { return ToServer }
func (*ToServerInventoryAction) Command() uint16            { return TOSERVER_INVENTORY_ACTION }
func (*ToServerInventoryAction) Direction() Direction       { return ToServer }
func (*ToServerChatMessage) Command() uint16                { return TOSERVER_CHAT_MESSAGE }
func (*ToServerChatMessage) Direction() Direction           { return ToServer }
func (*ToServerDamage) Command() uint16                     { return TOSERVER_DAMAGE }
func (*ToServerDamage) Direction() Direction                { return ToServer }
func (*ToServerPlayerItem) Command() uint16                 { return TOSERVER_PLAYERITEM }
func (*ToServerPlayerItem) Direction() Direction            { return ToServer }
func (*ToServerRespawnLegacy) Command() uint16              { return TOSERVER_RESPAWN_LEGACY }
func (*ToServerRespawnLegacy) Direction() Direction         { return ToServer }
func (*ToServerInteract) Command() uint16                   { return TOSERVER_INTERACT }
func (*ToServerInteract) Direction() Direction              { return ToServer }
func (*ToServerRemovedSounds) Command() uint16              { return TOSERVER_REMOVED_SOUNDS }
func (*ToServerRemovedSounds) Direction() Direction         { return ToServer }
func (*ToServerNodemetaFields) Command() uint16             { return TOSERVER_NODEMETA_FIELDS }
func (*ToServerNodemetaFields) Direction() Direction        { return ToServer }
func (*ToServerInventoryFields) Command() uint16            { return TOSERVER_INVENTORY_FIELDS }
func (*ToServerInventoryFields) Direction() Direction       { return ToServer }
func (*ToServerRequestMedia) Command() uint16               { return TOSERVER_REQUEST_MEDIA }
func (*ToServerRequestMedia) Direction() Direction          { return ToServer }
func (*ToServerHaveMedia) Command() uint16                  { return TOSERVER_HAVE_MEDIA }
func (*ToServerHaveMedia) Direction() Direction             { return ToServer }
func (*ToServerClientReady) Command() uint16                { return TOSERVER_CLIENT_READY }
func (*ToServerClientReady) Direction() Direction           { return ToServer }
func (*ToServerFirstSRP) Command() uint16                   { return TOSERVER_FIRST_SRP }
func (*ToServerFirstSRP) Direction() Direction              { return ToServer }
func (*ToServerSRPBytesA) Command() uint16                  { return TOSERVER_SRP_BYTES_A }
func (*ToServerSRPBytesA) Direction() Direction             { return ToServer }
func (*ToServerSRPBytesM) Command() uint16                  { return TOSERVER_SRP_BYTES_M }
func (*ToServerSRPBytesM) Direction() Direction             { return ToServer }
func (*ToServerUpdateClientInfo) Command() uint16           { return TOSERVER_UPDATE_CLIENT_INFO }
func (*ToServerUpdateClientInfo) Direction() Direction      { return ToServer }

func init() {
	register[ToClientHello](ToClient, "TOCLIENT_HELLO")
	register[ToClientAuthAccept](ToClient, "TOCLIENT_AUTH_ACCEPT")
	register[ToClientAcceptSudoMode](ToClient, "TOCLIENT_ACCEPT_SUDO_MODE")
	register[ToClientDenySudoMode](ToClient, "TOCLIENT_DENY_SUDO_MODE")
	register[ToClientAccessDenied](ToClient, "TOCLIENT_ACCESS_DENIED")
	register[ToClientBlockData](ToClient, "TOCLIENT_BLOCKDATA")
	register[ToClientAddNode](ToClient, "TOCLIENT_ADDNODE")
	register[ToClientRemoveNode](ToClient, "TOCLIENT_REMOVENODE")
	register[ToClientInventory](ToClient, "TOCLIENT_INVENTORY")
	register[ToClientTimeOfDay](ToClient, "TOCLIENT_TIME_OF_DAY")
	register[ToClientCSMRestrictionFlags](ToClient, "TOCLIENT_CSM_RESTRICTION_FLAGS")
	register[ToClientPlayerSpeed](ToClient, "TOCLIENT_PLAYER_SPEED")
	register[ToClientMediaPush](ToClient, "TOCLIENT_MEDIA_PUSH")
	register[ToClientChatMessage](ToClient, "TOCLIENT_CHAT_MESSAGE")
	register[ToClientActiveObjectRemoveAdd](ToClient, "TOCLIENT_ACTIVE_OBJECT_REMOVE_ADD")
	register[ToClientActiveObjectMessages](ToClient, "TOCLIENT_ACTIVE_OBJECT_MESSAGES")
	register[ToClientHP](ToClient, "TOCLIENT_HP")
	register[ToClientMovePlayer](ToClient, "TOCLIENT_MOVE_PLAYER")
	register[ToClientAccessDeniedLegacy](ToClient, "TOCLIENT_ACCESS_DENIED_LEGACY")
	register[ToClientFov](ToClient, "TOCLIENT_FOV")
	register[ToClientDeathscreenLegacy](ToClient, "TOCLIENT_DEATHSCREEN_LEGACY")
	register[ToClientMedia](ToClient, "TOCLIENT_MEDIA")
	register[ToClientNodeDef](ToClient, "TOCLIENT_NODEDEF")
	register[ToClientAnnounceMedia](ToClient, "TOCLIENT_ANNOUNCE_MEDIA")
	register[ToClientItemDef](ToClient, "TOCLIENT_ITEMDEF")
	register[ToClientPlaySound](ToClient, "TOCLIENT_PLAY_SOUND")
	register[ToClientStopSound](ToClient, "TOCLIENT_STOP_SOUND")
	register[ToClientPrivileges](ToClient, "TOCLIENT_PRIVILEGES")
	register[ToClientInventoryFormspec](ToClient, "TOCLIENT_INVENTORY_FORMSPEC")
	register[ToClientDetachedInventory](ToClient, "TOCLIENT_DETACHED_INVENTORY")
	register[ToClientShowFormspec](ToClient, "TOCLIENT_SHOW_FORMSPEC")
	register[ToClientMovement](ToClient, "TOCLIENT_MOVEMENT")
	register[ToClientSpawnParticle](ToClient, "TOCLIENT_SPAWN_PARTICLE")
	register[ToClientAddParticleSpawner](ToClient, "TOCLIENT_ADD_PARTICLESPAWNER")
	register[ToClientSpawnParticleBatch](ToClient, "TOCLIENT_SPAWN_PARTICLE_BATCH")
	register[ToClientCamera](ToClient, "TOCLIENT_CAMERA")
	register[ToClientHudAdd](ToClient, "TOCLIENT_HUDADD")
	register[ToClientHudRm](ToClient, "TOCLIENT_HUDRM")
	register[ToClientHudChange](ToClient, "TOCLIENT_HUDCHANGE")
	register[ToClientHudSetFlags](ToClient, "TOCLIENT_HUD_SET_FLAGS")
	register[ToClientHudSetParam](ToClient, "TOCLIENT_HUD_SET_PARAM")
	register[ToClientBreath](ToClient, "TOCLIENT_BREATH")
	register[ToClientSetSky](ToClient, "TOCLIENT_SET_SKY")
	register[ToClientOverrideDayNightRatio](ToClient, "TOCLIENT_OVERRIDE_DAY_NIGHT_RATIO")
	register[ToClientLocalPlayerAnimations](ToClient, "TOCLIENT_LOCAL_PLAYER_ANIMATIONS")
	register[ToClientEyeOffset](ToClient, "TOCLIENT_EYE_OFFSET")
	register[ToClientDeleteParticleSpawner](ToClient, "TOCLIENT_DELETE_PARTICLESPAWNER")
	register[ToClientCloudParams](ToClient, "TOCLIENT_CLOUD_PARAMS")
	register[ToClientFadeSound](ToClient, "TOCLIENT_FADE_SOUND")
	register[ToClientUpdatePlayerList](ToClient, "TOCLIENT_UPDATE_PLAYER_LIST")
	register[ToClientModChannelMsg](ToClient, "TOCLIENT_MODCHANNEL_MSG")
	register[ToClientModChannelSignal](ToClient, "TOCLIENT_MODCHANNEL_SIGNAL")
	register[ToClientNodemetaChanged](ToClient, "TOCLIENT_NODEMETA_CHANGED")
	register[ToClientSetSun](ToClient, "TOCLIENT_SET_SUN")
	register[ToClientSetMoon](ToClient, "TOCLIENT_SET_MOON")
	register[ToClientSetStars](ToClient, "TOCLIENT_SET_STARS")
	register[ToClientMovePlayerRel](ToClient, "TOCLIENT_MOVE_PLAYER_REL")
	register[ToClientSRPBytesSB](ToClient, "TOCLIENT_SRP_BYTES_S_B")
	register[ToClientFormspecPrepend](ToClient, "TOCLIENT_FORMSPEC_PREPEND")
	register[ToClientMinimapModes](ToClient, "TOCLIENT_MINIMAP_MODES")
	register[ToClientSetLighting](ToClient, "TOCLIENT_SET_LIGHTING")
	register[ToServerInit](ToServer, "TOSERVER_INIT")
	register[ToServerInit2](ToServer, "TOSERVER_INIT2")
	register[ToServerModChannelJoin](ToServer, "TOSERVER_MODCHANNEL_JOIN")
	register[ToServerModChannelLeave](ToServer, "TOSERVER_MODCHANNEL_LEAVE")
	register[ToServerModChannelMsg](ToServer, "TOSERVER_MODCHANNEL_MSG")
	register[ToServerPlayerPos](ToServer, "TOSERVER_PLAYERPOS")
	register[ToServerGotBlocks](ToServer, "TOSERVER_GOTBLOCKS")
	register[ToServerDeletedBlocks](ToServer, "TOSERVER_DELETEDBLOCKS")
	register[ToServerInventoryAction](ToServer, "TOSERVER_INVENTORY_ACTION")
	register[ToServerChatMessage](ToServer, "TOSERVER_CHAT_MESSAGE")
	register[ToServerDamage](ToServer, "TOSERVER_DAMAGE")
	register[ToServerPlayerItem](ToServer, "TOSERVER_PLAYERITEM")
	register[ToServerRespawnLegacy](ToServer, "TOSERVER_RESPAWN_LEGACY")
	register[ToServerInteract](ToServer, "TOSERVER_INTERACT")
	register[ToServerRemovedSounds](ToServer, "TOSERVER_REMOVED_SOUNDS")
	register[ToServerNodemetaFields](ToServer, "TOSERVER_NODEMETA_FIELDS")
	register[ToServerInventoryFields](ToServer, "TOSERVER_INVENTORY_FIELDS")
	register[ToServerRequestMedia](ToServer, "TOSERVER_REQUEST_MEDIA")
	register[ToServerHaveMedia](ToServer, "TOSERVER_HAVE_MEDIA")
	register[ToServerClientReady](ToServer, "TOSERVER_CLIENT_READY")
	register[ToServerFirstSRP](ToServer, "TOSERVER_FIRST_SRP")
	register[ToServerSRPBytesA](ToServer, "TOSERVER_SRP_BYTES_A")
	register[ToServerSRPBytesM](ToServer, "TOSERVER_SRP_BYTES_M")
	register[ToServerUpdateClientInfo](ToServer, "TOSERVER_UPDATE_CLIENT_INFO")
}
