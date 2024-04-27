package main

import (
  "encoding/json"
  "fmt"
  "github.com/gorilla/websocket"
  "github.com/pion/webrtc/v3"
  "log"
  "net/http"
  "os"
  "time"
)

type RTCSessionDescription struct {
  Type string `json:"type"`
  Sdp  string `json:"sdp"`
}

type Candidate struct {
  Candidate     string `json:"candidate"`
  SdpMid        string `json:"sdpMid"`
  SdpMLineIndex int    `json:"sdpMLineIndex"`
}

type Message struct {
  Type      string                 `json:"type"`
  Offer     *RTCSessionDescription `json:"offer,omitempty"`
  Answer    *RTCSessionDescription `json:"answer,omitempty"`
  Candidate *Candidate             `json:"candidate,omitempty"`
  From      string                 `json:"from"`
  To        string                 `json:"to"`
}

var upgrader = websocket.Upgrader{
  CheckOrigin: func(r *http.Request) bool {
    return true
  },
}

func main() {
  http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
    // CORS ヘッダの設定
    w.Header().Set("Access-Control-Allow-Origin", "*") // 本番環境では具体的なドメイン名を指定することが推奨されます
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
      fmt.Println("Error upgrading WebSocket:", err)
      return
    }
    defer conn.Close()

    peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
    if err != nil {
      fmt.Println("Error creating PeerConnection:", err)
      return
    }
    defer peerConnection.Close()

    // Opusファイルを読み込み
    fileData, err := os.ReadFile("/Users/shogo/ss/web_rtc_test/test_0427_01/server/opus128.opus")
    if err != nil {
      log.Println("Failed to read opus file:", err)
      return
    }

    // ここでメディアトラックを作成
    // Prepare to echo back audio
    localTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{
      MimeType:    "audio/opus",
      ClockRate:   48000,
      Channels:    2,
      SDPFmtpLine: "minptime=10;useinbandfec=1",
    }, "audio", "server-audio-track")
    if err != nil {
      log.Println("Error creating local audio track:", err)
      return
    }

    // Add this remoteTrack to the connection
    if _, err = peerConnection.AddTrack(localTrack); err != nil {
      fmt.Println("Error adding local remoteTrack:", err)
      return
    }

    peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
      fmt.Printf("Connection State has changed: %s\n", state.String())
      if state == webrtc.PeerConnectionStateConnected {
        log.Printf("Detect Connected!")
        //log.Printf("Detect Connected ! send candidate")
        //// candidateメッセージを送信
        //message := &Message{
        //  Type: "candidate",
        //  Candidate: &Candidate{
        //    Candidate:     "candidate",
        //    SdpMid:        "sdpMid",
        //    SdpMLineIndex: 0,
        //  },
        //  From: "user2",
        //}
        //b, err := json.Marshal(message)
        //if err != nil {
        //  log.Fatal(err)
        //}
        //conn.WriteMessage(websocket.TextMessage, b)
      }

      // クライアントへの音声送信
      go func() {
        ticker := time.NewTicker(20 * time.Millisecond) // 20msごとに送信
        for range ticker.C {
          //if err := localTrack.WriteRTP(media.Sample{Data: fileData, Duration: time.Second}); err != nil {
          //  log.Println("Error writing audio data:", err)
          //  break
          //}
        }
      }()
    })

    // Monitoring sender state (replace with actual monitoring/logging as needed)
    peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
      log.Printf("ICE Connection State has changed: %s\n", state.String())
    })

    peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
      if candidate == nil {
        log.Println("candidate is nil")
        return // 候補がなくなったことを示す。
      }
      log.Println("candidate is not nil")
      candidateData := candidate.ToJSON()
      if candidateData.Candidate == "" {
        return // 空の候補は無視する。
      }

      // SdpMid または SdpMLineIndex が nil である可能性の対処
      sdpMid := "default" // 適切なデフォルト値を設定
      log.Printf("*candidateData.SDPMid: %+v", *candidateData.SDPMid)
      if candidateData.SDPMid != nil {
        sdpMid = *candidateData.SDPMid
      }
      //if sdpMid == "" {
      //  sdpMid = "default"
      //}

      sdpMLineIndex := 0 // 通常、デフォルトとして 0 を使用
      log.Printf("*candidateData.SDPMLineIndex: %+v", *candidateData.SDPMLineIndex)
      if candidateData.SDPMLineIndex != nil {
        sdpMLineIndex = int(*candidateData.SDPMLineIndex)
      }

      message := Message{
        Type: "candidate",
        Candidate: &Candidate{
          Candidate:     candidateData.Candidate,
          SdpMid:        sdpMid,
          SdpMLineIndex: sdpMLineIndex,
        },
      }
      b, err := json.Marshal(message)
      if err != nil {
        log.Println("Error marshaling candidate:", err)
        return
      }
      conn.WriteMessage(websocket.TextMessage, b)
      log.Println("send candidate.")
    })

    // go func()で10秒おきにpcのStatsを出力
    go func() {
      ticker := time.NewTicker(10 * time.Second)
      for range ticker.C {
        status := peerConnection.ConnectionState().String()
        log.Printf("Report: %s", status)
      }
    }()

    // Handle tracks
    peerConnection.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
      fmt.Println("Track received:", remoteTrack.Kind().String())

      log.Println("remoteTrack.Codec()")
      log.Println(remoteTrack.Codec())
      log.Println("remoteTrack.ID()")
      log.Println(remoteTrack.ID())
      log.Println("remoteTrack.StreamID()")
      log.Println(remoteTrack.StreamID())

      //// Prepare to echo back audio
      //localTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: "audio/opus"}, "audio", "some-random-id")
      //if err != nil {
      //  fmt.Println("Error creating local remoteTrack:", err)
      //  return
      //}
      //
      //// Add this remoteTrack to the connection
      //if _, err = peerConnection.AddTrack(localTrack); err != nil {
      //  fmt.Println("Error adding local remoteTrack:", err)
      //  return
      //}

      // Echo back the audio
      if remoteTrack.Kind() == webrtc.RTPCodecTypeAudio {
        //go func() {
        //  buf := make([]byte, 1500)
        //  for {
        //    i, _, readErr := remoteTrack.Read(buf)
        //    if readErr != nil {
        //      fmt.Println("Error reading from track:", readErr)
        //      return
        //    }
        //    if _, writeErr := localTrack.Write(buf[:i]); writeErr != nil {
        //      fmt.Println("Error writing to track:", writeErr)
        //      return
        //    }
        //  }
        //}()
        go func() {
          for {
            i, _, readErr := remoteTrack.Read(fileData)
            if readErr != nil {
              fmt.Println("Error reading from track:", readErr)
              return
            }
            if _, writeErr := localTrack.Write(fileData[:i]); writeErr != nil {
              fmt.Println("Error writing to track:", writeErr)
              return
            }
          }
        }()
      }
      //// Write the buffer to the track
      //_, writeErr := localTrack.Write(fileData)
      //if writeErr != nil {
      //  panic(writeErr)
      //}
    })

    //peerConnection.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
    // fmt.Printf("Received a remote track: %s\n", remoteTrack.ID())
    //
    // localTrackHelper, err := webrtc.NewTrackLocalStaticRTP(remoteTrack.Codec().RTPCodecCapability, remoteTrack.ID(), remoteTrack.StreamID())
    // if err != nil {
    //   fmt.Println(err)
    // }
    //
    // rtpBuf := make([]byte, 1400)
    // go func() {
    //   for {
    //     i, _, readErr := remoteTrack.Read(rtpBuf)
    //     if readErr != nil {
    //       panic(readErr)
    //     }
    //
    //     //// Log the received data size
    //     //fmt.Printf("Received %d bytes of data\n", i)
    //     //
    //     //// Introduce a delay to simulate network latency
    //     //time.Sleep(500 * time.Millisecond) // 500 milliseconds delay
    //
    //     if _, err = localTrackHelper.Write(rtpBuf[:i]); err != nil && !errors.Is(err, io.ErrClosedPipe) {
    //       panic(err)
    //     }
    //   }
    // }()
    //
    // _, err = peerConnection.AddTrack(localTrackHelper)
    // if err != nil {
    //   return
    // }
    //})

    // Handle incoming WebSocket messages
    for {
      _, message, err := conn.ReadMessage()
      if err != nil {
        log.Println("Error reading WebSocket message:", err)
        break
      }
      log.Printf("recv: %s", message)

      var msg map[string]interface{}
      if err := json.Unmarshal(message, &msg); err != nil {
        log.Println("json decode:", err)
        continue
      }

      switch msg["type"] {
      case "offer":
        log.Println("Received offer")
      case "join":
        log.Println("Received join")
      case "candidate":
        log.Println("Received candidate")
      case "answer":
        log.Println("Received answer")
      default:
        log.Println("Unknown message type")
      }

      //// Unmarshal the JSON into webrtc.SessionDescription
      //var offer webrtc.SessionDescription
      //if err := json.Unmarshal(message, &offer); err != nil {
      //  log.Println("Error unmarshaling SDP offer:", err)
      //  continue
      //}

      // 'offer' キーが存在するかチェック
      offerData, ok := msg["offer"]
      if !ok {
        log.Println("No offer field in message")
        continue
      }

      // オファーをデコード
      offer := webrtc.SessionDescription{}
      if err := Decode(offerData, &offer); err != nil {
        log.Println(err)
        return
      }

      // Set the remote description to the incoming offer
      if err := peerConnection.SetRemoteDescription(offer); err != nil {
        log.Println("Error setting remote description:", err)
        continue
      }

      // Create and send an answer
      answer, err := peerConnection.CreateAnswer(nil)
      if err != nil {
        log.Println("Error creating answer:", err)
        continue
      }

      if err := peerConnection.SetLocalDescription(answer); err != nil {
        log.Println("Error setting local description:", err)
        continue
      }

      m := &Message{
        Type:   "answer",
        Answer: &RTCSessionDescription{Type: "answer", Sdp: answer.SDP},
        From:   "user2",
        To:     "user1",
      }
      b, err := json.Marshal(m)
      if err != nil {
        log.Fatal(err)
      }
      if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
        log.Println("Error sending answer via WebSocket:", err)
        continue
      }
    }
  })

  http.ListenAndServe(":8080", nil)
}

func Decode(data interface{}, v interface{}) error {
  // 型アサーションを使用して、安全にデータ型をチェック
  mapData, ok := data.(map[string]interface{})
  if !ok {
    return fmt.Errorf("decode error: data is not a map[string]interface{}")
  }

  // マップデータからJSONにエンコード
  jsonData, err := json.Marshal(mapData)
  if err != nil {
    return fmt.Errorf("decode error: %v", err)
  }

  // JSONデータを目的の構造体にデコード
  if err := json.Unmarshal(jsonData, v); err != nil {
    return fmt.Errorf("decode error: %v", err)
  }

  return nil
}
