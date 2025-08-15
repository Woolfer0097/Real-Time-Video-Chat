"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

export default function WaitingPage() {
  const router = useRouter();
  const [availableUsers, setAvailableUsers] = useState<number>(0);
  const [error, setError] = useState<string>("");
  const [matchFound, setMatchFound] = useState<string | null>(null); // Room ID when matched
  const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8000";

  useEffect(() => {
    const userId = typeof window !== "undefined" ? localStorage.getItem("user_id") : null;
    if (!userId) {
      router.push("/");
      return;
    }

    // Start polling for available users and potential matches
    const pollInterval = setInterval(async () => {
      try {
        // Check for available users count
        const countResponse = await fetch(`${API_BASE}/api/match/available-count`);
        if (countResponse.ok) {
          const countData = await countResponse.json();
          setAvailableUsers(countData.count);
        }

        // Check if we got matched
        const matchResponse = await fetch(`${API_BASE}/api/match/check?user_id=${encodeURIComponent(userId)}`);
        if (matchResponse.ok) {
          const matchData = await matchResponse.json();
          if (matchData.matched && matchData.room_id) {
            setMatchFound(matchData.room_id);
            return;
          }
        }
      } catch (err) {
        console.error("Polling error:", err);
        setError("Connection error. Please refresh the page.");
      }
    }, 2000); // Poll every 2 seconds

    // Cleanup on unmount
    return () => clearInterval(pollInterval);
  }, [router, API_BASE]);

  const handleCancel = () => {
    const userId = localStorage.getItem("user_id");
    if (userId) {
      // Mark user as unavailable
      fetch(`${API_BASE}/api/users/${userId}/availability`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ available: false }),
      }).catch(console.error);
    }
    router.push("/matching");
  };

  const handleStartVideoCall = () => {
    if (matchFound) {
      router.push(`/room/${matchFound}`);
    }
  };

  return (
    <div className="min-h-screen w-full font-body bg-gradient-to-br from-background to-background muted flex items-center justify-center p-4">
      <div className="bg-card rounded-2xl shadow-xl p-8 max-w-md w-full text-center relative overflow-hidden">
        {/* Gradient overlay */}
        <div className="absolute inset-0 bg-gradient-to-br from-blue/5 to-purple/5 pointer-events-none"></div>
        
        <div className="relative z-10">
          <div className="mb-8">
            <div className="w-16 h-16 bg-gradient-to-br from-blue/20 to-purple/20 rounded-full flex items-center justify-center mx-auto mb-4">
              {matchFound ? (
                <svg className="w-8 h-8 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              ) : (
                <svg className="w-8 h-8 text-blue animate-pulse" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              )}
            </div>
            <h1 className="text-2xl font-bold text-foreground mb-2">
              {matchFound ? "Match found!" : "Looking for a match..."}
            </h1>
            <p className="text-muted-foreground">
              {matchFound 
                ? "We found someone for you! Click below to start your video call." 
                : "We're finding someone for you to practice with"
              }
            </p>
          </div>

          <div className="mb-8">
            <div className="bg-gradient-to-r from-blue/10 to-purple/10 rounded-lg p-4 mb-4 border border-blue/20">
              <div className="text-sm text-blue mb-1">Available users</div>
              <div className="text-2xl font-bold bg-gradient-to-r from-blue to-purple bg-clip-text text-transparent">{availableUsers}</div>
            </div>
            
            <div className="flex space-x-2 justify-center">
              <div className="w-2 h-2 bg-gradient-to-r from-blue to-purple rounded-full animate-bounce"></div>
              <div className="w-2 h-2 bg-gradient-to-r from-blue to-purple rounded-full animate-bounce" style={{ animationDelay: '0.1s' }}></div>
              <div className="w-2 h-2 bg-gradient-to-r from-blue to-purple rounded-full animate-bounce" style={{ animationDelay: '0.2s' }}></div>
            </div>
          </div>

          {error && (
            <div className="mb-6 p-3 bg-red-50 border border-red-200 rounded-lg">
              <p className="text-red-600 text-sm">{error}</p>
            </div>
          )}

          <div className="space-y-3">
            {matchFound ? (
              <>
                <button
                  onClick={handleStartVideoCall}
                  className="w-full bg-gradient-to-r from-green-500 to-green-600 hover:from-green-600 hover:to-green-700 text-white font-medium py-3 px-4 rounded-lg transition-all duration-200 shadow-lg hover:shadow-xl"
                >
                  Start Video Call
                </button>
                <button
                  onClick={handleCancel}
                  className="w-full bg-muted hover:bg-muted/80 text-muted-foreground font-medium py-3 px-4 rounded-lg transition-colors"
                >
                  Cancel
                </button>
              </>
            ) : (
              <button
                onClick={handleCancel}
                className="w-full bg-muted hover:bg-muted/80 text-muted-foreground font-medium py-3 px-4 rounded-lg transition-colors"
              >
                Cancel
              </button>
            )}
          </div>

          <div className="mt-6 text-xs text-muted-foreground">
            <p>This usually takes 10-30 seconds</p>
            <p>Make sure your camera and microphone are ready!</p>
          </div>
        </div>
      </div>
    </div>
  );
}
