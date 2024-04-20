package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/pion/webrtc/v3"
)

func main() {
	// WebRTC APIの設定
	m := webrtc.MediaEngine{}
	m.RegisterDefaultCodecs()
	api := webrtc.NewAPI(webrtc.WithMediaEngine(&m))

	// WebRTCの設定: PeerConnectionの作成
	config := webrtc.Configuration{}
	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		log.Fatalf("Failed to create PeerConnection: %v", err)
	}
	defer peerConnection.Close()

	// オーディオトラックの追加
	audioTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: "audio/opus"}, "audio", "pion")
	if err != nil {
		log.Fatalf("Failed to create audio track: %v", err)
	}
	_, err = peerConnection.AddTrack(audioTrack)
	if err != nil {
		log.Fatalf("Failed to add audio track: %v", err)
	}

	// イベントハンドラの設定
	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("ICE Connection State has changed to %s\n", state.String())
	})

	// シグナリング: Offer SDPの作成
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		log.Fatalf("Failed to create offer: %v", err)
	}
	err = peerConnection.SetLocalDescription(offer)
	if err != nil {
		log.Fatalf("Failed to set local description: %v", err)
	}

	// HTTPサーバーを起動して、SDP情報を交換
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "SDP: %s", peerConnection.LocalDescription().SDP)
		log.Println("Offer SDP has been sent to the client.")
	})

	// HTTPサーバーの起動
	log.Println("HTTP server started on 0.0.0.0:8080")
	if err := http.ListenAndServe("0.0.0.0:8080", nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
