# StreamSaver
**StreamSaver** helps you download your favorite video for offline viewing by simply sharing the video through a companion app.

## 🌟 Features
- Written in Go and powered by yt-dlp and ffmpeg. This backend server is designed to robustly handle video download requests.
- Companion iOS App: Share a video link directly from your browser or YouTube, and the backend server takes care of the rest.
- Progress Tracking: Always be in the loop! Track download progress in real-time directly from the companion app.
- In-App playback: No need to look for downloaded videos. Play your downloaded videos directly from within the app.

## 🚀 Getting Started

### Prerequisite
- A Docker installation
- A folder with sufficient capacity and R/W permissions for downloaded videos and converted streaming content.
### Using Docker
It is very easy to bring up the server using Docker. Everything is preconfigured by Dockerfile and docker-compose.yml.
- Clone this repository
- Navigate to the project folder
- Edit docker-compose.yml and change volume mapping and port mapping as needed. By default, port **1718** is used for StreamSaver and port **1719** is for nginx streaming server. 
- Build and run the Docker image: `docker-compose up -d --build`

## Dependencies
- gorilla mux
- yt-dlp and ffmpeg for download and media file manipulation
- nginx for streaming hls content