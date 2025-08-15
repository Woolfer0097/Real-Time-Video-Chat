# WebRTC Signaling Server

This is a WebRTC signaling server implementation in Go that handles peer-to-peer video chat connections. The server manages:
- WebRTC signaling messages
- room management
- provides STUN/TURN server configuration
- and have been dockerized to be able to use as microservice and be independent of frontend