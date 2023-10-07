# Use the official Golang image as a base image
FROM golang:1.21

# Install ffmpeg and python
RUN apt-get update && apt-get install -y ffmpeg python3-full python3-pip git patch 

# Build yt-dlp
WORKDIR /usr/src/yt-dlp
RUN git clone https://github.com/yt-dlp/yt-dlp.git

RUN python3 -m venv venv
ENV PATH="/usr/src/yt-dlp/venv/bin:$PATH"
WORKDIR /usr/src/yt-dlp/yt-dlp

# Apply manual patch to the source file
COPY ./patch/ytdlp_common_py.diff .
RUN patch ./yt_dlp/downloader/common.py ytdlp_common_py.diff
RUN python3 -m pip install -U pyinstaller -r requirements.txt 
RUN python3 devscripts/make_lazy_extractors.py
RUN python3 pyinst.py
RUN cp dist/* /usr/local/bin/yt-dlp

# Set the working directory in the container to /app
WORKDIR /usr/src/dl_backend

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Change to the directory containing the main.go file and build the Go app
COPY . .
RUN cd cmd/dl_backend && go build -v -o /usr/local/bin/dl_backend

# Copy the binary file from the builder stage
RUN mkdir -p /root/.yt-dlp
COPY ./configs/ytdlp_config /root/.yt-dlp/config

# Make sure ffmpeg is installed
RUN ffmpeg -version

# Make sure yt-dlp is installed
RUN yt-dlp --version

# Test download the first video uploaded to youtube
RUN dl_backend&
RUN curl -iX POST http://localhost:1718 "https://youtu.be/jNQXAC9IVRw?si=kNdzUW09bzCDT6Iw"

# Give it some time to complete the download
RUN sleep 5 
RUN ./tests/test_first_download.sh

# Set the command to run when starting the container
CMD ["dl_backend"]

