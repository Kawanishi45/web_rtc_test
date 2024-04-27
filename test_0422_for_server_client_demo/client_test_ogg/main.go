package main

import (
  "fmt"
  "github.com/pion/mediadevices"
  "github.com/pion/mediadevices/pkg/codec"
  "github.com/pion/mediadevices/pkg/prop"
  "github.com/pion/webrtc/v3"
  "os"
  "os/signal"
  "syscall"
)

func main() {
  // WebRTCの設定
  config := webrtc.Configuration{
    ICEServers: []webrtc.ICEServer{
      {
        URLs: []string{"stun:stun.l.google.com:19302"},
      },
    },
  }

  // APIオブジェクトの作成
  api := webrtc.NewAPI()

  // メディアデバイスの設定
  codecSelector := mediadevices.NewCodecSelector(
    mediadevices.WithAudioEncoders(codec.AudioEncoderBuilder(func(sampleRate int, channelNum int) (codec.AudioEncoder, error) {
      return codec.NewPCMAudioEncoder(sampleRate, channelNum), nil
    })),
  )

  // ピアコネクションの作成
  peerConnection, err := api.NewPeerConnection(config)
  if err != nil {
    panic(err)
  }

  // 音声トラックの取得と追加
  audioConstraints := func(c *mediadevices.MediaTrackConstraints) {
    c.MediaConstraints = prop.MediaConstraints{
      AudioConstraints: prop.AudioConstraints{
        SampleRate:   prop.IntConstraint(int(48000)),
        ChannelCount: prop.IntConstraint(int(2)),
      },
    }
  }

  mediaStream, err := mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
    Audio: audioConstraints,
    Codec: codecSelector,
  })
  if err != nil {
    panic(err)
  }

  for _, track := range mediaStream.GetTracks() {
    _, err = peerConnection.AddTrack(track)
    if err != nil {
      panic(err)
    }
  }

  // Offerを作成
  offer, err := peerConnection.CreateOffer(nil)
  if err != nil {
    panic(err)
  }
  err = peerConnection.SetLocalDescription(offer)
  if err != nil {
    panic(err)
  }

  // ここでSDPを表示または他のピアに送信
  fmt.Printf("Offer SDP:\n%s\n", offer.SDP)

  // シグナリング用のチャネルや処理をここに追加

  // 終了処理の設定
  c := make(chan os.Signal, 1)
  signal.Notify(c, os.Interrupt, syscall.SIGTERM)
  <-c
  fmt.Println("終了します...")

  // ピアコネクションのクローズ
  peerConnection.Close()
}
