# गो का ऑफिशियल इमेज लें
FROM golang:1.22-alpine

# सिस्टम में ffmpeg, python3, py3-pip, curl और ca-certificates इनस्टॉल करें
RUN apk add --no-cache ffmpeg python3 py3-pip curl ca-certificates && \
    curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp && \
    chmod a+rx /usr/local/bin/yt-dlp

# वर्किंग डायरेक्टरी सेट करें
WORKDIR /app

# गो मॉड्यूल्स कॉपी करें
COPY go.mod ./
RUN go mod download

# बाकी सारा कोड कॉपी करें
COPY . .

# गो ऐप को कंपाइल करें
RUN go build -o tsb-server .

# पोर्ट एक्सपोज़ करें
EXPOSE 8080

# सर्वर रन करें
CMD ["./tsb-server"]