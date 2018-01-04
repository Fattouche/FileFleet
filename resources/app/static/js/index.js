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

function sendMessage(message){
	astilectron.sendMessage(message, function(message) {
        document.getElementById("app-message").innerHTML = message.payload
    })

}

function rcvFile() {
	var message = validateInput(checkFile=false)
	if(!message) return
	sendMessage(message)
}

function sendFile() {
	var message = validateInput(checkFile=true)
	if(!message) return
	sendMessage(message)
}
