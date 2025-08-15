"use client";

import { useEffect, useRef, useState } from "react";
import { useParams, useRouter } from "next/navigation";

export default function RoomPage() {
  const { roomId } = useParams<{ roomId: string }>();
  const router = useRouter();
  const localVideoRef = useRef<HTMLVideoElement | null>(null);
  const remoteVideoRef = useRef<HTMLVideoElement | null>(null);
  const pcRef = useRef<RTCPeerConnection | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const [connected, setConnected] = useState(false);
  const isInitiatorRef = useRef(false);
  const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8000";

  useEffect(() => {
    let isMounted = true;
    const start = async () => {
      try {
        const stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true });
        if (!isMounted) return;
        if (localVideoRef.current) {
          localVideoRef.current.srcObject = stream;
        }

        const pc = new RTCPeerConnection({
          iceServers: [{ urls: ["stun:stun.l.google.com:19302"] }],
        });
        pcRef.current = pc;
        stream.getTracks().forEach((t) => pc.addTrack(t, stream));

        pc.ontrack = (ev) => {
          console.log("Received remote track:", ev);
          if (remoteVideoRef.current) {
            // attach first stream
            if (remoteVideoRef.current.srcObject !== ev.streams[0]) {
              remoteVideoRef.current.srcObject = ev.streams[0];
              console.log("Remote video stream attached");
            }
          }
        };

        pc.onicecandidate = (ev) => {
          if (ev.candidate && wsRef.current) {
            wsRef.current.send(
              JSON.stringify({ type: "ice_candidate", data: ev.candidate })
            );
          }
        };

        pc.onconnectionstatechange = () => {
          console.log("Connection state changed:", pc.connectionState);
          if (pc.connectionState === "connected") {
            setConnected(true);
            console.log("WebRTC connection established!");
          } else if (pc.connectionState === "failed" || pc.connectionState === "disconnected") {
            setConnected(false);
            console.log("WebRTC connection failed or disconnected");
          }
        };

        pc.oniceconnectionstatechange = () => {
          console.log("ICE connection state changed:", pc.iceConnectionState);
          if (pc.iceConnectionState === "connected" || pc.iceConnectionState === "completed") {
            setConnected(true);
            console.log("WebRTC ICE connection established!");
          } else if (pc.iceConnectionState === "failed" || pc.iceConnectionState === "disconnected") {
            setConnected(false);
            console.log("WebRTC ICE connection failed or disconnected");
          }
        };

        const wsUrl = API_BASE.replace("http", "ws") + "/webrtc";
        const ws = new WebSocket(wsUrl);
        wsRef.current = ws;

        ws.onopen = async () => {
          ws.send(JSON.stringify({ type: "join_room", room_id: roomId }));
        };

        ws.onmessage = async (event) => {
          const msg = JSON.parse(event.data);
          console.log("WebRTC message received:", msg);
          
          switch (msg.type) {
            case "room_joined": {
              console.log("Room joined, is_initiator:", msg.data?.is_initiator);
              console.log("Room data:", msg.data);
              // Store initiator status but don't create offer yet
              // Wait for peer_joined event if we're the initiator
              if (msg.data && msg.data.is_initiator) {
                isInitiatorRef.current = true;
                console.log("I am the initiator, waiting for peer to join");
              } else {
                isInitiatorRef.current = false;
                console.log("Waiting for offer as non-initiator");
              }
              break;
            }
            case "peer_joined": {
              console.log("Peer joined:", msg.data);
              console.log("Am I initiator?", isInitiatorRef.current);
              // If I'm the initiator and a peer just joined, create offer
              if (isInitiatorRef.current) {
                console.log("Creating offer as initiator since peer joined");
                const offer = await pc.createOffer();
                await pc.setLocalDescription(offer);
                ws.send(
                  JSON.stringify({ type: "offer", data: offer })
                );
                console.log("Offer sent");
              } else {
                console.log("Not initiator, waiting for offer");
              }
              break;
            }
            case "offer": {
              console.log("Received offer, creating answer");
              await pc.setRemoteDescription(new RTCSessionDescription(msg.data));
              const answer = await pc.createAnswer();
              await pc.setLocalDescription(answer);
              ws.send(JSON.stringify({ type: "answer", data: answer }));
              console.log("Answer sent");
              break;
            }
            case "answer": {
              console.log("Received answer, setting remote description");
              await pc.setRemoteDescription(new RTCSessionDescription(msg.data));
              console.log("Remote description set from answer");
              break;
            }
            case "ice_candidate": {
              try {
                console.log("Adding ICE candidate");
                await pc.addIceCandidate(new RTCIceCandidate(msg.data));
              } catch (err) {
                console.error("Failed to add ICE candidate:", err);
              }
              break;
            }
            case "error": {
              console.error("WebRTC error:", msg.error);
              // basic error surface
              alert(msg.error || "Signaling error");
              break;
            }
          }
        };

        ws.onclose = () => setConnected(false);
      } catch (err) {
        console.error(err);
        alert("Failed to start camera/microphone");
        router.push("/matching");
      }
    };

    start();
    return () => {
      isMounted = false;
      wsRef.current?.close();
      pcRef.current?.getSenders().forEach((s) => {
        try { s.track?.stop(); } catch {}
      });
      pcRef.current?.close();
    };
  }, [API_BASE, roomId, router]);

  return (
    <div className="min-h-screen w-full font-body p-4">
      <h1 className="font-heading text-2xl mb-4">Room: {roomId}</h1>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="rounded-xl overflow-hidden bg-black aspect-video">
          <video ref={localVideoRef} autoPlay playsInline muted className="w-full h-full object-cover" />
        </div>
        <div className="rounded-xl overflow-hidden bg-black aspect-video">
          <video ref={remoteVideoRef} autoPlay playsInline className="w-full h-full object-cover" />
        </div>
      </div>
      <div className="mt-4 text-sm text-[--color-muted]">
        {connected ? "Connected" : "Connecting..."}
      </div>
    </div>
  );
}


