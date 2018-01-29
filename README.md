# P2P_File_Transferer
This repository was made to allow for simple and fast filetransfer without needing to upload or pay for any third party service. Since this is just used for file transfer, a peer will only connect to at most one other peer. All filetransfer logic was done using google's quic protocol and more specifically https://github.com/lucas-clemente/quic-go.

![](resources/images/logo.png)

*note - as of right now, the rendevousz server that is responsible for connecting peers is hosted on an EC2 instance. 

## Getting Started Usage
1. Check out the releases and download the correct version for your operating system!

## Getting Started Development
1. Clone the repository `git@github.com:Fattouche/FileFleet.git`
2. I recommend using VSCode for golang, however, any terminal will do.
3. Download quic `go get "github.com/lucas-clemente/quic-go" `
4. Download the GUI bundler `go get -u github.com/asticode/go-astilectron-bundler/...`
5. Build the application `astilectron-bundler -v`
6. You might get golang errors, just run `go get <url>` for any errors

## Code Components

1. Rendevousz Server - The peers communicate through this to find eachother and excahnge initial information.
2. Peer - These will communicate on behalf of the user to transfer the file.

## Code Overview

1. The peers will connect to the rendevousz server and exchange information.
2. The peers will attempt to punch a hole through eachothers NAT by sending udp packets to the peers public ip.
3. If the peers can recieve eachothers udp packets, they can establish a quic connection using the other peers public ip.
4. If they cannot recieve eachothers udp packets, they will attempt to establish a quic connection using their private IPs.
5. If once again they cannot connect privately, maybe due to access point isolation, they peers will once again connect to the server and transfer through the server.

## License

This project is licensed under the GNU General Public License v3.0 License - see the [LICENSE.md](LICENSE) file for details
