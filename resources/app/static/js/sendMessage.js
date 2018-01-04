// This will wait for the astilectron namespace to be ready
document.addEventListener('astilectron-ready', function() {
    // This will send a message to GO
    astilectron.sendMessage({name: "event.name", payload: "hello"}, function(message) {
        console.log("received " + message.payload)
    });
})