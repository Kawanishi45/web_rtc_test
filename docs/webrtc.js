let localVideo = document.getElementById('localVideo');
let remoteVideo = document.getElementById('remoteVideo');
let startAs1Button = document.getElementById('startAs1Button');
let startAs2Button = document.getElementById('startAs2Button');
let offerAs1Button = document.getElementById('offerAs1Button');

let localStream;
let remoteStream;
let peerConnection;
const serverUrl = 'ws://localhost:8080/ws'; // 適切なサーバーURLを使用してください
// const serverUrl = 'ws://10.16.4.135:8080/ws'; // 適切なサーバーURLを使用してください
// const serverUrl = 'wss://e3ec-217-178-107-208.ngrok-free.app/ws'; // 適切なサーバーURLを使用してください
let ws;

const configuration = {
    iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
};

// 接続の設定と開始
function setupWebSocket(userId) {
    ws = new WebSocket(serverUrl+'?userId='+userId);
    ws.onopen = function() {
        console.log('WebSocket connection established');
        createPeerConnection();
        send({ type: 'join', from: userId }); // 'user1' は適宜変更
    };
    ws.onerror = function(event) {
        console.error('WebSocket error observed:', event);
    };
    ws.onclose = function(event) {
        console.log('WebSocket connection closed:', event);
    };
    ws.onmessage = handleMessage;
}

startAs1Button.onclick = async () => {
    startAs1Button.disabled = true;
    localStream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true });
    localVideo.srcObject = localStream;

    setupWebSocket('user1');
};

startAs2Button.onclick = async () => {
    startAs2Button.disabled = true;
    localStream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true });
    localVideo.srcObject = localStream;

    setupWebSocket('user2');
};

offerAs1Button.onclick = async () => {
    // Offerを生成
    let offer = await peerConnection.createOffer();
    await peerConnection.setLocalDescription(offer);
    console.log('Offer created and set as local description:', offer);
    // ICE Gatheringが完了したらOfferを送信
    peerConnection.onicecandidate = event => {
        if (event.candidate === null) { // ICE Gatheringが完了
            console.log('ICE Gathering Complete', peerConnection.localDescription);
            send({ type: 'offer', offer: peerConnection.localDescription, to: 'user2', from: 'user1' }); // Offerを送信
        }
    };
};

peerConnection.onconnectionstatechange = function(event) {
    console.log(`Connection State: ${peerConnection.connectionState}`);
    if (peerConnection.connectionState === 'connected') {
        console.log('The connection has been fully established!');
    }
};

// ontrackイベントハンドラーを設定
peerConnection.ontrack = function(event) {
    console.log('Track received!');
    const mediaStream = event.streams[0];

    if (mediaStream && mediaStream.getAudioTracks().length > 0) {
        console.log('Audio track received');

        // オーディオトラックをHTMLのaudio要素にセットして再生
        const audioElement = document.createElement('audio');
        audioElement.srcObject = mediaStream;
        audioElement.play();

        // 再生を確認
        audioElement.onplaying = () => {
            console.log('Audio is playing');
        };

        // エラーを監視
        audioElement.onerror = (e) => {
            console.error('Error playing audio:', e);
        };

        // 音声データを確認するためのユーザーインターフェイスに追加
        document.body.appendChild(audioElement);
    }

    const audioContext = new AudioContext();
    const mediaStreamSource = audioContext.createMediaStreamSource(event.streams[0]);
    const analyzer = audioContext.createAnalyser();

    mediaStreamSource.connect(analyzer);
    analyzer.fftSize = 2048;
    const bufferLength = analyzer.frequencyBinCount;
    const dataArray = new Uint8Array(bufferLength);

    function analyzeAudio() {
        requestAnimationFrame(analyzeAudio);
        analyzer.getByteTimeDomainData(dataArray);
        // ここで dataArray を使用して音声データの分析を行う
        console.log(dataArray); // 波形データの生の値をログに出力
    }

    analyzeAudio();
};

function createPeerConnection() {
    peerConnection = new RTCPeerConnection(configuration);
    localStream.getTracks().forEach(track => {
        peerConnection.addTrack(track, localStream);
    });

    peerConnection.ontrack = event => {
        remoteStream = event.streams[0];
        remoteVideo.srcObject = remoteStream;
    };

    peerConnection.onicecandidate = event => {
        if (event.candidate) {
            send({ type: 'candidate', candidate: event.candidate, from: 'user1' }); // ICE Candidateを送信
        }
    };
}

function send(message) {
    ws.send(JSON.stringify(message));
    console.log('Message sent:', message);
}

async function handleMessage(message) {
    console.log('handleMessage');

    let data = JSON.parse(message.data);
    console.log('Message received:', data);

    switch (data.type) {
        case 'offer':
            console.log('Received offer:', data.offer);
            await peerConnection.setRemoteDescription(new RTCSessionDescription(data.offer));
            let answer = await peerConnection.createAnswer();
            await peerConnection.setLocalDescription(answer);
            send({ type: 'answer', answer: answer, to: data.from });
            break;
        case 'answer':
            console.log('Received answer:', data.answer);
            // const sdpType = data[data.type].type; // Ensure this exists and is correct
            //             const sdp = data[data.type].sdp; // Ensure SDP exists
            //             if (sdpType && (sdpType === 'offer' || sdpType === 'answer' || sdpType === 'pranswer' || sdpType === 'rollback') && sdp) {
            //                 const sessionDescription = new RTCSessionDescription({
            //                     type: sdpType,
            //                     sdp: sdp
            //                 });
            //                 await peerConnection.setRemoteDescription(sessionDescription);
            //                 if (sdpType === 'offer') {
            //                     const answer = await peerConnection.createAnswer();
            //                     await peerConnection.setLocalDescription(answer);
            //                     send({ type: 'answer', answer: answer, to: data.from });
            //                 }
            //             } else {
            //                 console.error('Invalid SDP type or missing SDP:', sdpType);
            //             }
            await peerConnection.setRemoteDescription(new RTCSessionDescription(data.answer));
            // console.log('Received answer:', data);
            // await peerConnection.setRemoteDescription(new RTCSessionDescription(data));
            console.log('setRemoteDescription');
            break;
        // case 'candidate':
        //     console.log('Received ICE candidate:', data.candidate);
        //     try {
        //         await peerConnection.addIceCandidate(new RTCIceCandidate(data.candidate));
        //     } catch (e) {
        //         console.error('Error adding received ice candidate:', e);
        //     }
        //     break;
        case 'candidate':
            if (data.candidate) {
                console.log('Received ICE candidate:', data.candidate);
                try {
                    const iceCandidate = new RTCIceCandidate({
                        candidate: data.candidate.candidate,
                        sdpMid: data.candidate.sdpMid,
                        sdpMLineIndex: data.candidate.sdpMLineIndex
                    });
                    await peerConnection.addIceCandidate(iceCandidate);
                } catch (e) {
                    console.error('Error adding received ice candidate:', e);
                }
            }
            break;
    }
}
