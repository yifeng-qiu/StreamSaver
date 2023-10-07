# 🎥 StreamSaver
Conveniently download your favorite videos with **StreamSaver**. 

## 🌟 Features
- Written in Go and powered by yt-dlp and ffmpeg. This backend server is designed to robustly handle video download requests.
- Companion iOS App: Share a video link directly from your browser or YouTube, and the backend server takes care of the rest.
- Progress Tracking: Always be in the loop! Track download progress in real-time directly from the companion app.
- In-App playback: No need to go into your server to find the downloaded videos. Play your downloaded videos directly from within the app.

## 🚀 Getting Started

### Prerequisite
- A folder with sufficient capacity and R/W permissions for downloaded videos
### Using Docker
Using Docker to set up the server is the easiest. 
- Clone this repository
- Navigate to the project folder: cd StreamSaver
- Edit docker-compose.yml and change the volume mapping as needed. 
- Build and run the Docker image: `docker-compose up -d --build`

### Without Docker
Setup StreamSaver Server
Clone this repository: git clone https://github.com/yourusername/StreamSaver.git
Navigate to the project folder: cd StreamSaver
Run: go build to compile the project.
Start the server: ./StreamSaver
Install the Companion App
Follow steps from the Docker section


- Install and configure the companion App. See separate instructions. 
- Share a video link using the app's share extension.
- Wait for the download to complete.
- Play and enjoy your video!