# Go का ऑफिशियल इमेज लें
FROM golang:1.22-alpine

# सिस्टम में yt-dlp और ffmpeg इनस्टॉल करें ताकि सर्वर पर वीडियो प्रोसेस हो सके
RUN apk add --no-cache ffmpeg python3 curl && \
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
RUN go build -o tsb-server main.go

# पोर्ट एक्सपोज़ करें
EXPOSE 8080

# सर्वर रन करें
CMD ["./tsb-server"]