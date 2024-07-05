const formatter = Intl.NumberFormat('en', { notation: 'compact' });
function nString(value) {
	return formatter.format(value)
}

window.onload = function() {
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

	stats.onopen = function() {
		console.log('Connected to stats.');
	};
	stats.onerror = function (error) {
		console.error("An unknown error occured", error);
	};
	
	stats.onclose = function (event) {
		console.log("Server closed connection", event);
	}

	stats.onmessage = function(event) {
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

	socket.onopen = function() {
		console.log('Connected to icoflut.');
	};
	socket.onerror = function (error) {
		console.error("An unknown error occured", error);
	};
	
	socket.onclose = function (event) {
		console.log("Server closed connection", event);
	}

	socket.onmessage = function(event) {
		urlCreator = window.URL || window.webkitURL;
		favicon.href = urlCreator.createObjectURL(event.data);
	}
}
