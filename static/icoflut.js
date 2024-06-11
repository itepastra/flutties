function nString(value) {
    var newValue = value;
    if (value >= 1000) {
        var suffixes = ["", "k", "M", "B", "T", "Q"];
        var suffixNum = Math.floor( (""+value).length/3 );
        var shortValue = '';
        for (var precision = 2; precision >= 1; precision--) {
            shortValue = parseFloat( (suffixNum != 0 ? (value / Math.pow(1000,suffixNum) ) : value).toPrecision(precision));
            var dotLessShortValue = (shortValue + '').replace(/[^a-zA-Z 0-9]+/g,'');
            if (dotLessShortValue.length <= 2) { break; }
        }
        if (shortValue % 1 != 0)  shortValue = shortValue.toFixed(1);
        newValue = shortValue+suffixes[suffixNum];
    }
    return newValue;
}

window.onload = function() {
	var favicon = document.getElementById("favicon");

	var client = document.getElementById("clientCounter");
	var pixel = document.getElementById("pixelCounter");

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
		client.innerText = "Currently there are " + nString(obj.c) + " clients connected."
		pixel.innerText = "So far " + nString(obj.p) + " pixels of the main canvas have been modified, and " + nString(obj.i) + " of the icon"
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
