# Naive Firewall

It allows multiple clients to connect to the server and send messages either to specific clients or broadcast to all clients. Additional features include blacklisting IPs and ports, blocking specific clients, and controlling broadcast reception.

## Features

- **Broadcast Messaging**: Send a message to all connected clients.
- **Direct Messaging**: Send a message to a specific client by selecting their ID.
- **Client Blocking**: Block communication with specific clients.
- **Broadcast Control**: Toggle broadcast message reception.
- **IP/Port Blacklisting**: Prevent connections from specific IPs or ports.
- **Automatic Reconnection**: Clients will automatically attempt to reconnect if the server is temporarily unavailable.


### Clone the repository

```bash
git clone https://github.com/yourusername/tcp-chat-app.git
cd tcp-chat-app
