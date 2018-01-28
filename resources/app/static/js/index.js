window.onload = function () {
	asticode.loader.init();
	asticode.modaler.init();
	asticode.notifier.init();
}

document.addEventListener('astilectron-ready', function () {
	astilectron.onMessage(function (message) {
		mes = message.name
		switch (message.name) {
			case "Error":
				console.log("ERROR: " + message.payload)
				document.getElementById("postToApp").innerHTML = ""
				document.getElementById("app-message").innerHTML = message.payload
				break
			case "Connected":
				console.log("connected")
				document.getElementById("postToApp").innerHTML = ""
				document.getElementById("postToApp").innerHTML = '<i class="fa fa-refresh fa-spin fa-3x fa-fw"></i><br><br>Transfering file'
				break
			case "Server":
				console.log("Connected through server, transfer may take longer")
				document.getElementById("postToApp").innerHTML = ""
				document.getElementById("postToApp").innerHTML = '<i class="fa fa-refresh fa-spin fa-3x fa-fw"></i><br><br>Could not connect to peer<br><br>Transfering file through server<br><br>This may take a while'
				break
			case "Finished":
				console.log("FINISHED: " + message.payload)
				document.getElementById("postToApp").innerHTML = ""
				document.getElementById("app-message").innerHTML = "Transfer complete!"
			case "about":
				let c = document.createElement("div");
				c.innerHTML = message.payload;
				asticode.modaler.setContent(c);
				asticode.modaler.show();
		}
		return { payload: "payload" };
	})
})


function validateInput(checkFile) {
	var Peer1 = document.getElementById("Peer1").value
	var Peer2 = document.getElementById("Peer2").value
	var returnMessage = {}

	if (Peer1.length == 0 || Peer2.length == 0 || Peer1.length > 50 || Peer2.length > 50 || Peer1 === Peer2) {
		console.log("Error, invalid input")
		document.getElementById("error-message").innerHTML = "Peer names must be unique and between 1 and 50 characters"
		return false
	}
	returnMessage["Peer1"] = Peer1
	returnMessage["Peer2"] = Peer2

	if (checkFile) {
		var file = document.getElementById("FileName").files[0].path
		if (file.length == 0) {
			console.log("Error, invalid file input")
			document.getElementById("error-message").innerHTML = "Please choose a file"
			return false
		}
		returnMessage["FileName"] = file
	} else {
		var directory = document.getElementById("fileInput").files[0].path
		if (directory.length == 0) {
			console.log("Error, invalid directory")
			document.getElementById("error-message").innerHTML = "Please choose a save location"
			return false
		}
		returnMessage["Directory"] = directory
	}
	document.getElementById("error-message").innerHTML = ""
	console.log(returnMessage)
	return returnMessage
}

function sendMessage(input) {
	document.getElementById("postToApp").innerHTML = "Connecting to peer..."
	document.getElementById("postToApp").removeAttribute("onclick")

	let payloadString = JSON.stringify(input)
	let mes = { name: "info", payload: payloadString }

	astilectron.sendMessage({ name: "info", payload: payloadString }, function (message) {
		console.log("RECIEVED: " + message.payload)
	})
}

function rcvFile() {
	var input = validateInput(checkFile = false)
	if (!input) return
	sendMessage(input)
}

function sendFile() {
	var input = validateInput(checkFile = true)
	if (!input) return
	sendMessage(input)
}
