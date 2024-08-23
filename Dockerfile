# 1. Go 언어 빌드 스테이지
FROM golang:1.23-alpine3.20 AS builder

# 2. 작업 디렉토리 설정
WORKDIR /app

# 3. 모듈 파일 복사 및 종속성 설치
COPY go.mod go.sum ./
RUN go mod download

# 4. 소스 코드 복사
COPY . .

# 5. 애플리케이션 빌드
RUN go build -o server .

# 6. 실행 스테이지
FROM alpine:latest

# 7. CA 인증서 설치 (PostgreSQL과 같은 서비스의 SSL 연결을 위해 필요할 수 있음)
RUN apk --no-cache add ca-certificates

# 8. 작업 디렉토리 설정
WORKDIR /root/

# 9. 빌드된 바이너리와 필요한 파일 복사
COPY --from=builder /app/server .
COPY --from=builder /app/.index .index

# 10. 사용할 포트 노출
EXPOSE 8080

# 11. Docker build-time ARG 설정
ARG POSTGRES_CONN_ARG

# 12. 환경 변수로 POSTGRES_CONN 설정
ENV POSTGRES_CONN=${POSTGRES_CONN_ARG}

# 13. 컨테이너 시작 시 실행할 명령
CMD ["./server"]
