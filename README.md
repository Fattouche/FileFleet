# P2P_File_Transferer
This repository was made to allow for simple and fast filetransfer without needing to upload or pay for any third party service. Since this is just used for file transfer, a peer will only connect to at most one other peer. All filetransfer logic was done using google's quic protocol and more specifically https://github.com/lucas-clemente/quic-go.

![](Images/sharing.png)

*note - as of right now, the rendevousz server that is responsible for connecting peers is hosted on an EC2 instance. Therefore this service is not always available.

## Getting Started Usage
1. Usage requires that one peer acts as a reciever and one acts as a sender.
2. To initialize as the sender of the file: `go run [your_name] [reciever_name] [file]`
3. To initialize as the reciever of the file: `go run [your_name] [reciever_name]
4. If you want to host your own server : `go run main.go` and change the IP that the peer connects to in peer.go.

## Getting Started Development
1. Clone the repository `git@github.com:Fattouche/P2P_File_Transferer.git`
2. I recommend using VSCode for golang, however, any terminal will do.
3. `go get "github.com/lucas-clemente/quic-go" `

## Code Overview

Due to the nature of peer to peer connections, the code requires 3 different components. The first component is a general rendevous server which is required to collect information from connecting peers and send the information back to the peers. Having this centralized server allows the connecting peers to avoid confusions with NAT tables because they already know the information of the server. Since most routers implement DHCP, it was required that the peers connect to the server using UDP to ensure that they can both dial and listen on the same IP that the server will be sending to the other peer. Once the peers have exchanged information through the server, the peers can connect to eachother via a direct connection. Using the same UDP connection as used to connect to the rendevous server, the peers attempt to punch a hole through eachothers NAT in their respective routers. If a holepunch cannot be established, the peers try to send the file within the network, otherwise they send outside the network.

## License

This project is licensed under the GNU General Public License v3.0 License - see the [LICENSE.md](LICENSE) file for details
