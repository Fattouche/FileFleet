# P2P_File_Transferer
This repository was made to allow for simple and fast filetransfer without needing to upload or pay for any third party service. Since this is just used for file transfer, a peer will only connect to at most one other peer. 

*note - as of right now, the rendevousz server that is responsible for connecting peers is hosted on an EC2 instance. Therefore this service is not always available.

## Getting Started Usage
1. Usage requires that one peer acts as a reciever and one acts as a sender.
2. To initialize as the sender of the file: `go run [your_name] [reciever_name] [file]`
3. To initialize as the reciever of the file: `go run [your_name] [reciever_name]

## Getting Started Development
1. Clone the repository `git@github.com:Fattouche/P2P_File_Transferer.git`
2. I recommend using VSCode for golang, however, any terminal will do.

## Code Overview

Due to the nature of peer to peer connections, the code requires 3 different components. The first component is a general rendevous server which is required to collect information from connecting peers and send the information back to the peers. Having this centralized server allows the connecting peers to avoid confusions with NAT tables because they already know the information of the server. Since most routers implement DHCP, it was required that the peers connect to the server using UDP to ensure that they can both dial and listen on the same IP that the server will be sending to the other peer. Once the peers have exchanged information through the server, the peers can connect to eachother via a direct connection. Before attempting to deal with any UDP holepunching or NAT tables, the peers attempt to connect via TCP inside the same network, a timeout is used if a TCP connection cannot be established, otherwise the file is transfered using tCP. If the peers cannot establish a TCP connection, they begin sending UDP packets to eachother until both peers recieve a UDP packet from the other peer, at which point the file transfer will begin using the entry that was just created in the respective routers. Due to the nature of UDP, a simple reliable protocol was created which just keeps track of the number of bytes recieved by the receiving peer.

## License

This project is licensed under the GNU General Public License v3.0 License - see the [LICENSE.md](LICENSE.md) file for details