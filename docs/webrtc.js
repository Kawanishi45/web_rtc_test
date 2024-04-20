let localVideo = document.getElementById('localVideo');
let remoteVideo = document.getElementById('remoteVideo');
let startAs1Button = document.getElementById('startAs1Button');
let startAs2Button = document.getElementById('startAs2Button');
let offerAs1Button = document.getElementById('offerAs1Button');

let localStream;
let remoteStream;
let peerConnection;
// const serverUrl = 'ws://10.16.4.135:8080/ws'; // 適切なサーバーURLを使用してください
const serverUrl = 'ws://e3ec-217-178-107-208.ngrok-free.app/ws'; // 適切なサーバーURLを使用してください
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
                console.error('Error adding received ice candidate:', e);
            }
            break;
    }
}
