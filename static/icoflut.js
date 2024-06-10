
window.onload = function() {
	var favicon = document.getElementById("favicon");
	var faviconSize = 32;
	var canvas = document.createElement("canvas");
	var context = canvas.getContext("2d");
	var img = document.createElement("img");
	img.src = favicon.href

	img.onload = function(){
		canvas.width = faviconSize;
		canvas.height = faviconSize;
		
		context.fillStyle="#000000";
		context.fillRect(0,0,canvas.width, canvas.height);

		favicon.href=canvas.toDataUrl("image/png");
	}
}
