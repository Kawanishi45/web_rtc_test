package main

import (
  "encoding/binary"
  "encoding/json"
  "fmt"
  "github.com/gorilla/websocket"
  "github.com/hajimehoshi/go-mp3"
  "github.com/hraban/opus"
  "github.com/pion/webrtc/v3"
  "github.com/pion/webrtc/v3/pkg/media"
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

    // MediaEngine に Opus コーデックを登録
    // MediaEngine を初期化して Opus を登録
    m := webrtc.MediaEngine{}
    if err := m.RegisterCodec(webrtc.RTPCodecParameters{
      RTPCodecCapability: webrtc.RTPCodecCapability{
        MimeType:    webrtc.MimeTypeOpus,
        ClockRate:   48000,
        Channels:    2,
        SDPFmtpLine: "minptime=10;useinbandfec=1",
      },
      PayloadType: 111,
    }, webrtc.RTPCodecTypeAudio); err != nil {
      log.Fatal("Failed to register codec: ", err)
    }

    // API オブジェクトの作成
    api := webrtc.NewAPI(webrtc.WithMediaEngine(&m))

    peerConnection, err := api.NewPeerConnection(webrtc.Configuration{
      ICEServers: []webrtc.ICEServer{
        {
          URLs: []string{"stun:stun.l.google.com:19302"},
        },
      },
    })
    if err != nil {
      fmt.Println("Error creating PeerConnection:", err)
      return
    }
    //defer peerConnection.Close()

    // 音声トラックの作成
    localTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "pion")
    if err != nil {
      log.Fatal(err)
    }
    //// 音声トラックの作成
    //    localTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "pion")
    //    if err != nil {
    //      log.Fatal(err)
    //    }

    // トラックをPeer connectionに追加
    _, err = peerConnection.AddTrack(localTrack)
    if err != nil {
      log.Fatal(err)
    }

    peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
      fmt.Printf("Connection State has changed: %s\n", state.String())
      if state == webrtc.PeerConnectionStateConnected {
        log.Printf("Detect Connected!")
      }
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
    //
    //// go func()で10秒おきにpcのStatsを出力
    //go func() {
    //  ticker := time.NewTicker(10 * time.Second)
    //  for range ticker.C {
    //    status := peerConnection.ConnectionState().String()
    //    log.Printf("Report: %s", status)
    //  }
    //}()

    // mp3ファイルを読み込む
    file, err := os.Open("/Users/shogo/ss/web_rtc_test/test_0427_01/server/hana.mp3")
    if err != nil {
      panic(err)
    }
    defer file.Close()

    // mp3デコーダの作成
    mp3Decoder, err := mp3.NewDecoder(file)
    if err != nil {
      panic(err)
    }

    // Opusエンコーダの設定
    opusEncoder, err := opus.NewEncoder(mp3Decoder.SampleRate(), 2, opus.AppAudio)
    if err != nil {
      panic(err)
    }

    // Handle tracks
    peerConnection.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
      fmt.Println("Track received:", remoteTrack.Kind().String())

      log.Println("remoteTrack.Codec()")
      log.Println(remoteTrack.Codec())
      log.Println("remoteTrack.ID()")
      log.Println(remoteTrack.ID())
      log.Println("remoteTrack.StreamID()")
      log.Println(remoteTrack.StreamID())

      // Echo back the audio
      if remoteTrack.Kind() == webrtc.RTPCodecTypeAudio {
        go func() {
          // ここでバッファの大きさを調整
          frameSize := 960 // Opusエンコーダのフレームサイズに依存
          pcmBuf := make([]int16, frameSize)
          byteBuf := make([]byte, frameSize*2) // 2 bytes per int16
          for {
            // mp3からPCMデータを読み込む（byte形式）
            var n int
            n, err = mp3Decoder.Read(byteBuf)
            if err != nil {
              break
            }

            // byteからint16に変換
            for i := 0; i < n/2; i++ {
              pcmBuf[i] = int16(binary.LittleEndian.Uint16(byteBuf[i*2 : (i*2)+2]))
            }

            // Opusにエンコード
            opusData := make([]byte, 1000)
            n, err = opusEncoder.Encode(pcmBuf[:n/2], opusData)
            if err != nil {
              break
            }

            // RTPパケットを送信
            err = localTrack.WriteSample(media.Sample{Data: opusData[:n], Duration: time.Millisecond * 20})
            if err != nil {
              break
            }
          }
        }()
      }
      //// Echo back the audio
      //if remoteTrack.Kind() == webrtc.RTPCodecTypeAudio {
      //  go func() {
      //    buf := make([]byte, 1500)
      //    for {
      //      i, _, readErr := remoteTrack.Read(buf)
      //      if readErr != nil {
      //        fmt.Println("Error reading from track:", readErr)
      //        return
      //      }
      //      if _, writeErr := localTrack.Write(buf[:i]); writeErr != nil {
      //        fmt.Println("Error writing to track:", writeErr)
      //        return
      //      }
      //    }
      //  }()
      //}
    })

    log.Println()

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

      log.Println("offer.Type", offer.Type)
      log.Println("offer.Type.String", offer.Type.String())
      //log.Println("offer.SDP", offer.SDP)

      offer.Type = webrtc.SDPTypeOffer
      log.Println("offer.Type", offer.Type)
      log.Println("offer.Type.String", offer.Type.String())

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
