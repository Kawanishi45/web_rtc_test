package main

import (
  "encoding/base64"
  "encoding/json"
  "fmt"
  "github.com/notedit/gstreamer-go"
  "github.com/pion/webrtc/v2"
  "log"
  "math/rand"
  "os"

  "github.com/pion/webrtc/v2/pkg/media"
)

func main() {
  offerb, err := base64.StdEncoding.DecodeString(os.Args[1])
  if err != nil {
    log.Fatalf("failed to read offer: %+v", err)
  }
  var offer webrtc.SessionDescription
  if err := json.Unmarshal(offerb, &offer); err != nil {
    log.Fatalf("failed to unmarshal offer: %+v", err)
  }

  mediaEngine := webrtc.MediaEngine{}
  if err := mediaEngine.PopulateFromSDP(offer); err != nil {
    log.Fatalf("failed to populate from SDP: %+v", err)
  }

  var payloadType uint8
  for _, videoCodec := range mediaEngine.GetCodecsByKind(webrtc.RTPCodecTypeVideo) {
    if videoCodec.Name == webrtc.H264 {
      payloadType = videoCodec.PayloadType
      break
    }
  }
  if payloadType == 0 {
    log.Fatal("Remote peer does not support H264")
  }

  api := webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine))
  pc, err := api.NewPeerConnection(webrtc.Configuration{
    ICEServers: []webrtc.ICEServer{
      {
        URLs: []string{"stun:stun.l.google.com:19302"},
      },
    },
  })
  if err != nil {
    log.Fatalf("failed to new peeer connection: %+v", err)
  }
  pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
    log.Printf("peer state change: %v", state)
  })

  videoTrack, err := pc.NewTrack(payloadType, rand.Uint32(), "video", "pion")
  if err != nil {
    log.Fatalf("failed to new track: %+v", err)
  }
  if _, err = pc.AddTrack(videoTrack); err != nil {
    log.Fatalf("failed to add track: %+v", err)
  }

  // video streaming via GStreamer
  go func() {
    pipeline, err := gstreamer.New("videotestsrc is-live=true ! video/x-raw,format=I420 ! x264enc ! appsink name=app")
    if err != nil {
      log.Printf("failed to new GStreamer pipeline: %+v", err)
      return
    }
    pipeline.Start()
    app := pipeline.FindElement("app")
    for data := range app.Poll() {
      if err := videoTrack.WriteSample(media.Sample{Data: data, Samples: 90000}); err != nil {
        log.Printf("failed to write video data: %+v", err)
      }
    }
  }()

  if err := pc.SetRemoteDescription(offer); err != nil {
    log.Fatalf("failed to set remote desc: %+v", err)
  }
  answer, err := pc.CreateAnswer(nil)
  if err != nil {
    log.Fatalf("failed to set create answer: %+v", err)
  }
  log.Printf("create answer")
  if err = pc.SetLocalDescription(answer); err != nil {
    log.Fatalf("failed to set local description: %+v", err)
  }

  answerb, err := json.Marshal(answer)
  if err != nil {
    log.Fatalf("failed to marshal answer: %+v", err)
  }
  answerb64 := base64.StdEncoding.EncodeToString(answerb)
  fmt.Println(answerb64)

  select {}
}
