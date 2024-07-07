const formatter = Intl.NumberFormat('en', { notation: 'compact' });
function nString(value) {
	return formatter.format(value)
}

var isDrawing = false;
var ws = undefined;

async function StartDrawing(e) {
	isDrawing = true;
	let ratioX = e.target.naturalWidth / e.target.offsetWidth;
	let ratioY = e.target.naturalHeight / e.target.offsetHeight;

	let domX = e.x + window.scrollX - e.target.offsetLeft;
	let domY = e.y + window.scrollY - e.target.offsetTop;

	let imgX = Math.floor(domX * ratioX);
	let imgY = Math.floor(domY * ratioY);

	let color = document.querySelector("input[name=color]:checked").value;
	let size = document.querySelector("input[name=size]:checked").value;

	if (typeof (ws) == WebSocket) {
		ws.send(JSON.stringify({ x: imgX, y: imgY, color: color, size: size }))
	} else {
		console.log("websocket needs to connect first")
	}
}

function StopDrawing() {
	isDrawing = false;
}

onpointermove = async function (e) {
	if (!isDrawing) { return; }
	let ratioX = e.target.naturalWidth / e.target.offsetWidth;
	let ratioY = e.target.naturalHeight / e.target.offsetHeight;

	let domX = e.x + window.scrollX - e.target.offsetLeft;
	let domY = e.y + window.scrollY - e.target.offsetTop;

	let imgX = Math.floor(domX * ratioX);
	let imgY = Math.floor(domY * ratioY);

	let color = document.querySelector("input[name=color]:checked").value;
	let size = document.querySelector("input[name=size]:checked").value;

	if (typeof ws !== 'undefined') {
		ws.send(JSON.stringify({ x: imgX, y: imgY, color: color, size: +size }))
	} else {
		console.log("websocket needs to connect first")
	}
};

window.onload = function () {
	var favicon = document.getElementById("favicon");

	var client = document.getElementById("clientCounter");
	var pixel = document.getElementById("pixelCounter");
	var pixelAvg = document.getElementById("pixelCounterAvg");
	var icon = document.getElementById("iconCounter");
	var iconAvg = document.getElementById("iconCounterAvg");

	var pixelQueue = [];
	var iconQueue = [];

	for (i = 0; i < 5; i++) {
		pixelQueue.push(0)
		iconQueue.push(0)
	}

	const socket = new WebSocket("/icoflut");
	const stats = new WebSocket("/stats");
	ws = stats;

	stats.onopen = function () {
		console.log('Connected to stats.');
	};
	stats.onerror = function (error) {
		console.error("An unknown error occured", error);
	};

	stats.onclose = function (event) {
		console.log("Server closed connection", event);
	}

	stats.onmessage = function (event) {
		const obj = JSON.parse(event.data);
		client.innerText = nString(obj.c)

		pixel.innerText = nString(obj.p)
		pixelQueue.push(obj.p)
		var old = pixelQueue.shift()
		pixelAvg.innerText = nString(obj.p - old)

		icon.innerText = nString(obj.i)
		iconQueue.push(obj.i)
		var old = iconQueue.shift()
		iconAvg.innerText = nString(obj.i - old)
	}

	socket.onopen = function () {
		console.log('Connected to icoflut.');
	};
	socket.onerror = function (error) {
		console.error("An unknown error occured", error);
	};

	socket.onclose = function (event) {
		console.log("Server closed connection", event);
	}

	socket.onmessage = function (event) {
		urlCreator = window.URL || window.webkitURL;
		favicon.href = urlCreator.createObjectURL(event.data);
	}
}
