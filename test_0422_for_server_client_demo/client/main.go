package main

import (
  "encoding/json"
  "flag"
  "fmt"
  "github.com/gordonklaus/portaudio"
  "github.com/gorilla/websocket"
  "github.com/pion/webrtc/v3"
  "github.com/pion/webrtc/v3/pkg/media"
  "log"
  "net/url"
  "os"
  "strconv"
  "time"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

var pc *webrtc.PeerConnection

var file *os.File

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

func main() {
  var err error
  flag.Parse()
  log.SetFlags(0)

  // 音声ファイルを開く
  //file, err = os.Open("/Users/shogo/ss/web_rtc_test/test_0422_for_server_client_demo/client/audio01_opus.ogg")
  file, err = os.Open("/Users/shogo/ss/web_rtc_test/test_0422_for_server_client_demo/client/output.opus")
  if err != nil {
    log.Fatalf("Failed to open opus file: %v", err)
  }
  defer func() {
    file.Close()
    log.Println("File closed")
  }()

  // PeerConnectionの作成
  pc, err = webrtc.NewPeerConnection(webrtc.Configuration{})
  if err != nil {
    log.Fatal(err)
  }

  // go func()で3秒おきにpcのStatsを出力
  go func() {
    ticker := time.NewTicker(3 * time.Second)
    for range ticker.C {
      status := pc.ConnectionState().String()
      log.Printf("Report: %s", status)
    }
  }()

  // WebSocketでサーバーとの接続を開始
  u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}

  // クエリパラメータを追加
  q := u.Query()
  q.Set("userId", "user2")
  u.RawQuery = q.Encode()

  log.Printf("connecting to %s", u.String())

  // WebSocket接続
  c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
  if err != nil {
    log.Fatal("dial:", err)
  }
  defer c.Close()

  // サーバーとのWebSocketコネクションを監視
  done := make(chan struct{})

  var message *Message
  var b []byte

  // candidateメッセージを送信
  message = &Message{
    Type: "candidate",
    Candidate: &Candidate{
      Candidate:     "candidate",
      SdpMid:        "sdpMid",
      SdpMLineIndex: 0,
    },
    From: "user2",
  }
  b, err = json.Marshal(message)
  if err != nil {
    log.Fatal(err)
  }
  c.WriteMessage(websocket.TextMessage, b)

  // joinメッセージを送信
  message = &Message{
    Type: "join",
    From: "user2",
  }
  b, err = json.Marshal(message)
  if err != nil {
    log.Fatal(err)
  }
  c.WriteMessage(websocket.TextMessage, b)

  go func() {
    defer close(done)
    for {
      _, message, err := c.ReadMessage()
      if err != nil {
        log.Println("read:", err)
        return
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
        // Offerを受信した場合、Answerを生成して送信
        handleOffer(msg, c)
      case "join":
        log.Println("Received join")
        // サーバーからのJoinメッセージを受信した場合、Offerを生成して送信
        createAndSendOffer(c)
      case "candidate":
        log.Println("Received candidate")
      case "answer":
        log.Println("Received answer")
      default:
        log.Println("Unknown message type")
      }
    }
  }()

  // ここでメインスレッドは、WebSocketの接続が終了するのを待ちます。
  <-done
}

type audioInput struct {
  buffer []int
}

func newAudioInput() *audioInput {
  return &audioInput{
    buffer: make([]int, 1024),
  }
}

func (a *audioInput) ProcessAudio(in []int) {
  copy(a.buffer, in)
}

// intのスライスをbyteのスライスに変換するヘルパー関数
func intToByte(data []int) []byte {
  result := make([]byte, len(data))
  for i, v := range data {
    result[i] = byte(v)
  }
  return result
}

func sound(pc *webrtc.PeerConnection) error {
  portaudio.Initialize()
  defer portaudio.Terminate()

  in := newAudioInput()
  stream, err := portaudio.OpenDefaultStream(1, 0, 44100, len(in.buffer), in.ProcessAudio)
  if err != nil {
    log.Fatalf("Failed to open default stream: %v", err)
  }
  defer stream.Close()

  err = stream.Start()
  if err != nil {
    log.Fatalf("Failed to start stream: %v", err)
  }

  // Opusのトラックを作成
  codec := webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}
  trackLocal, err := webrtc.NewTrackLocalStaticSample(codec, "audio", "pion")
  if err != nil {
    return err
  }

  // PeerConnectionにトラックを追加
  _, err = pc.AddTrack(trackLocal)
  if err != nil {
    return err
  }

  // 音声データを再生して送信
  go func() {
    for {
      // マイクからオーディオフレームを読み込む
      if err := stream.Read(); err != nil {
        log.Printf("Failed to read from mic: %v", err)
        return
      }

      // RTPパケットを作成して送信
      if err = trackLocal.WriteSample(media.Sample{Data: intToByte(in.buffer), Duration: time.Millisecond * 20}); err != nil {
        log.Printf("Failed to write opus sample: %v", err)
        return
      }

      // 適宜ウェイトを入れる
      time.Sleep(time.Millisecond * 20)
    }
  }()

  return nil
}

//func sound() error {
//  //// ファイルを開く
//  //file, err := os.Open("path/to/your/file.ogg")
//  //if err != nil {
//  //  return err
//  //}
//  //defer file.Close()
//
//  // Oggリーダーを作成
//  ogg, _, err := oggreader.NewWith(file)
//  if err != nil {
//    log.Fatalf("Failed to create ogg reader: %v", err)
//  }
//
//  // Opusのトラックを作成
//  codec := webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}
//  trackLocal, err := webrtc.NewTrackLocalStaticSample(codec, "audio", "pion")
//  if err != nil {
//    return err
//  }
//
//  log.Println("codec debug: ", codec.ClockRate, codec.MimeType, codec.SDPFmtpLine, codec.RTCPFeedback)
//
//  // PeerConnectionにトラックを追加
//  _, err = pc.AddTrack(trackLocal)
//  if err != nil {
//    return err
//  }
//
//  // 音声ファイルを再生して送信
//  go func() {
//    for {
//      // OggファイルからOpusパケットを読み込む
//      packet, _, err := ogg.ParseNextPage()
//      if err != nil {
//        if err == io.EOF {
//          // ファイルの開始位置に戻す
//          file.Seek(0, 0)
//          ogg, _, err = oggreader.NewWith(file)
//          if err != nil {
//            log.Printf("Failed to reset ogg reader: %v", err)
//            return
//          }
//          continue
//        }
//        log.Printf("Failed to parse ogg packet: %v", err)
//        return
//      }
//
//      // RTPパケットを作成して送信
//      if err = trackLocal.WriteSample(media.Sample{Data: packet, Duration: time.Millisecond * 20}); err != nil {
//        log.Printf("Failed to write opus sample: %v", err)
//        return
//      }
//
//      // 適宜ウェイトを入れる
//      time.Sleep(time.Millisecond * 20)
//    }
//  }()
//
//  return nil
//}

func handleOffer(msg map[string]interface{}, c *websocket.Conn) {
  // 'offer' キーが存在するかチェック
  offerData, ok := msg["offer"]
  if !ok {
    log.Println("No offer field in message")
    return
  }

  // オファーをデコード
  offer := webrtc.SessionDescription{}
  if err := Decode(offerData, &offer); err != nil {
    log.Println(err)
    return
  }

  err := sound(pc)
  if err != nil {
    log.Fatalf("Failed to send sound: %v", err)
  }

  // Offerをリモートディスクリプションとして設定
  if err := pc.SetRemoteDescription(offer); err != nil {
    log.Fatal(err)
  }

  // Answerを生成
  answer, err := pc.CreateAnswer(nil)
  if err != nil {
    log.Fatal(err)
  }

  // ローカルディスクリプションとしてAnswerを設定
  if err = pc.SetLocalDescription(answer); err != nil {
    log.Fatal(err)
  }

  // Answerをサーバーに送信
  sendAnswer(answer, c)
}

//func handleOffer(msg map[string]interface{}, c *websocket.Conn) {
//  // 'offer' キーが存在するかチェック
//  offerData, ok := msg["offer"]
//  if !ok {
//    log.Println("No offer field in message")
//    return
//  }
//
//  // オファーをデコード
//  offer := webrtc.SessionDescription{}
//  if err := Decode(offerData, &offer); err != nil {
//    log.Println(err)
//    return
//  }
//
//  //// PeerConnectionの作成
//  //peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
//  //if err != nil {
//  //  log.Fatal(err)
//  //}
//
//  // Offerをリモートディスクリプションとして設定
//  if err := pc.SetRemoteDescription(offer); err != nil {
//    log.Fatal(err)
//  }
//
//  // Answerを生成
//  answer, err := pc.CreateAnswer(nil)
//  if err != nil {
//    log.Fatal(err)
//  }
//
//  // ローカルディスクリプションとしてAnswerを設定
//  if err = pc.SetLocalDescription(answer); err != nil {
//    log.Fatal(err)
//  }
//
//  //pc = peerConnection
//
//  // Answerをサーバーに送信
//  sendAnswer(answer, c)
//}

func createAndSendOffer(c *websocket.Conn) {
  err := sound(pc)
  if err != nil {
    log.Fatalf("Failed to send sound: %v", err)
  }

  // Offerを生成
  offer, err := pc.CreateOffer(nil)
  if err != nil {
    log.Fatal(err)
  }

  // ローカルディスクリプションとしてOfferを設定
  if err := pc.SetLocalDescription(offer); err != nil {
    log.Fatal(err)
  }

  // Offerをサーバーに送信
  sendOffer(offer, c)
}

//func createAndSendOffer(c *websocket.Conn) {
//  //// PeerConnectionの作成
//  //peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
//  //if err != nil {
//  //  log.Fatal(err)
//  //}
//
//  // Offerを生成
//  offer, err := pc.CreateOffer(nil)
//  if err != nil {
//    log.Fatal(err)
//  }
//
//  // ローカルディスクリプションとしてOfferを設定
//  if err := pc.SetLocalDescription(offer); err != nil {
//    log.Fatal(err)
//  }
//
//  //pc = peerConnection
//
//  // Offerをサーバーに送信
//  sendOffer(offer, c)
//}

func sendOffer(offer webrtc.SessionDescription, c *websocket.Conn) {
  message := &Message{
    Type:  "offer",
    Offer: &RTCSessionDescription{Type: strconv.Itoa(int(offer.Type)), Sdp: offer.SDP},
    From:  "user2",
    To:    "user1",
  }
  b, err := json.Marshal(message)
  if err != nil {
    log.Fatal(err)
  }
  c.WriteMessage(websocket.TextMessage, b)
  log.Println("Sent offer")

  err = sound(pc)
  if err != nil {
    log.Fatalf("Failed to send sound: %v", err)
  }
}

func sendAnswer(answer webrtc.SessionDescription, c *websocket.Conn) {
  message := &Message{
    Type:   "answer",
    Answer: &RTCSessionDescription{Type: "answer", Sdp: answer.SDP},
    From:   "user2",
    To:     "user1",
  }
  b, err := json.Marshal(message)
  if err != nil {
    log.Fatal(err)
  }
  c.WriteMessage(websocket.TextMessage, b)
  log.Println("Sent answer")

  err = sound(pc)
  if err != nil {
    log.Fatalf("Failed to send sound: %v", err)
  }
}

//func Decode(data interface{}, v interface{}) {
//  mapData := data.(map[string]interface{})
//  jsonData, _ := json.Marshal(mapData)
//  json.Unmarshal(jsonData, v)
//}

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
