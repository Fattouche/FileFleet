function init() {
	asticode.loader.init();
    asticode.modaler.init();
    asticode.notifier.init();
}

function validateInput(checkFile){
	var Peer1 = document.getElementById("Peer1").value
	var Peer2 = document.getElementById("Peer2").value
	var returnMessage = {}

	if(Peer1.length == 0 || Peer2.length == 0 || Peer1.length > 50 || Peer2.length > 50 || Peer1 === Peer2) {
		console.log("Error, invalid input")
		document.getElementById("error-message").innerHTML = "Peer names must be unique and between 1 and 50 characters"
		return false
	} 
	returnMessage["Peer1"] = Peer1
	returnMessage["Peer2"] = Peer2

	if (checkFile){
		var file = document.getElementById("FileName").value
		if(file.length == 0) {
			console.log("Error, invalid file input")
			document.getElementById("error-message").innerHTML = "Please choose a file"
			return false
		}
		returnMessage["FileName"] = file
	}
	document.getElementById("error-message").innerHTML = ""
	console.log(returnMessage)
	return returnMessage
}

function sendMessage(input){
	document.getElementById("postToApp").innerHTML = "Connecting to peer..."
	document.getElementById("postToApp").removeAttribute("onclick")

	astilectron.sendMessage({name: "UserInput", payload: input}, function(message) {
		console.log(message.payload)
	})
}

function rcvMessage(){
	var mes
	document.addEventListener('astilectron-ready', function() {
		astilectron.onMessage(function(message) {
			mes = message.name
			switch (message.name) {
				case "error" :
					document.getElementById("postToApp").innerHTML = ""
					document.getElementById("app-message").innerHTML = message.message
					break
				case "connected":
					document.getElementById("postToApp").innerHTML = ""
					document.getElementById("postToApp").innerHTML = '<i class="fa fa-refresh fa-spin fa-3x fa-fw"></i>'
					break
				case "finished":
					document.getElementById("postToApp").innerHTML = ""
					document.getElementById("app-message").innerHTML = "Transfer complete!"
			}
		})
	})
	return mes
}

function rcvFile() {
	var input = validateInput(checkFile=false)
	if(!input) return
	sendMessage(input)
	while(rcvMessage() === "connected") {}
}

function sendFile() {
	var input = validateInput(checkFile=true)
	if(!input) return
	sendMessage(input)
	while(rcvMessage() === "connected") {}
}
