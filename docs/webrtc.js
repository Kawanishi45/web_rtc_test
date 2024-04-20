let localVideo = document.getElementById('localVideo');
let remoteVideo = document.getElementById('remoteVideo');
let startButton = document.getElementById('startButton');

let localStream;
let remoteStream;
let peerConnection;
// const serverUrl = 'ws://localhost:8080/ws'; // WebSocketサーバーのURL
const serverUrl = 'ws://10.16.4.135:8080/ws'; // WebSocketサーバーのURL
// const serverUrl = 'ws://9b37-217-178-107-208.ngrok-free.app/ws'; // WebSocketサーバーのURL
let ws;

const configuration = {
    iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
};

startButton.onclick = async () => {
    startButton.disabled = true;
    localStream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true });
    localVideo.srcObject = localStream;

    ws = new WebSocket(serverUrl);
    ws.onmessage = handleMessage;
    ws.onopen = () => {
        createPeerConnection();
        send({ type: 'join', from: 'user1' }); // 'user1' は適宜変更
    };
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
            const uniqueToken = generateToken();
            console.log("Generated unique token:", uniqueToken);
            send({ type: 'candidate', candidate: event.candidate, from: uniqueToken }); // 'user1' は適宜変更
        }
    };
}

function generateToken() {
    let array = new Uint32Array(1);
    window.crypto.getRandomValues(array);
    return array[0].toString(16);
}

async function handleMessage(message) {
    let data = JSON.parse(message.data);

    switch (data.type) {
        case 'offer':
            if (peerConnection.signalingState !== 'stable') {
                console.log('Offer dropped, signalingState is not stable');
                return;
            }
            await peerConnection.setRemoteDescription(new RTCSessionDescription(data.offer));
            let answer = await peerConnection.createAnswer();
            await peerConnection.setLocalDescription(answer);
            send({ type: 'answer', answer: answer, to: data.from });
            break;
        case 'answer':
            await peerConnection.setRemoteDescription(new RTCSessionDescription(data.answer));
            break;
        case 'candidate':
            try {
                await peerConnection.addIceCandidate(new RTCIceCandidate(data.candidate));
            } catch (e) {
                console.error('Error adding received ice candidate', e);
            }
            break;
    }
}

function send(message) {
    ws.send(JSON.stringify(message));
}
