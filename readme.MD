 /\     /\  
(  o   o  )  
 |   ^   |  
  \ --- /  
  /     \  
 /  🐾   \ ✋  
(         ) 

Image Generation WebSocket Client
A robust Go application that connects to an image generation service via WebSocket, processes requests, and handles image generation and distribution.
🌟 Features

WebSocket-based client with automatic reconnection
Secure TLS connection support
Multiple model support with dynamic configuration
Custom image generation parameters
Automatic session management
File upload capabilities
Pretty logging with colored output

🚀 Quick Start

Clone the repository:

bashCopygit clone https://github.com/yourusername/image-generation-client.git

Configure your settings in config.yaml:

yamlCopyserver:
  host: your.server.host
  port: "8080"
  passcode: "your-passcode"
api:
  host: api.host
  port: "8081"
  timeout: 240

Run the application:

bashCopygo run .
📋 Requirements

Go 1.21 or higher
Working image generation API endpoint
Valid SSL certificates (or configure to skip verification)

🔧 Configuration
The application uses a YAML configuration file with the following structure:
yamlCopyserver:
  host: string     # WebSocket server host
  port: string     # WebSocket server port
  passcode: string # Authentication passcode

api:
  host: string     # API server host
  port: string     # API server port
  timeout: int     # Request timeout in seconds

models:
  - name: string        # Model display name
    string: string      # Model identifier
    width: int         # Image width
    height: int        # Image height
    steps: int         # Generation steps
    cfgscale: float    # Configuration scale
    loras: string      # Optional LoRA settings
    loraweights: float # Optional LoRA weights
🔐 Security Features

TLS support for secure connections
Passcode-based authentication
Configurable timeouts
Secure file handling

📦 Project Structure
Copy.
├── main.go          # Application entry point
├── config.go        # Configuration handling
├── client.go        # HTTP client implementation
├── websocket.go     # WebSocket client implementation
├── logger.go        # Custom logger
├── config.yaml      # Configuration file
└── README.md        # This file
🐛 Debugging
The application includes a comprehensive logging system with different levels:

DEBUG: Detailed debugging information
INFO: General operational information
WARN: Warning messages for potentially problematic situations
ERROR: Error messages for serious problems

Logs are color-coded for better visibility:

🟣 DEBUG
🔵 INFO
🟡 WARN
🔴 ERROR

🤝 Contributing

Fork the repository
Create your feature branch (git checkout -b feature/amazing-feature)
Commit your changes (git commit -m 'Add some amazing feature')
Push to the branch (git push origin feature/amazing-feature)
Open a Pull Request

📜 License
This project is licensed under the MIT License - see the LICENSE file for details.
✨ Credits
ASCII cat art source: ASCII Art Archive
🔍 Known Issues

None at the moment. Please submit issues via GitHub's issue tracke