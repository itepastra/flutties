window.onload = function() {
	var favicon = document.getElementById("favicon");


	const socket = new WebSocket("ws://localhost:7792/icoflut");

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
