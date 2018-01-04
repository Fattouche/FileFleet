func main() {
	bootstrap.Run(bootstrap.Options{
		MessageHandler: handleMessages,	
	})
}

// handleMessages handles messages
func handleMessages(_ *astilectron.Window, m bootstrap.MessageIn) (payload interface{}, err error) {
	switch m.Name {
	case "event.name":
		// Unmarshal payload
		var s string
		if err = json.Unmarshal(m.Payload, &path); err != nil {
		    payload = err.Error()
		    return
		}
		payload = s + " world"
	}
	return
}