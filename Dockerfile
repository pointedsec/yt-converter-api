# Etapa de construcción
FROM golang:1.24-alpine AS builder

# Actualizar paquetes e instalar ffmpeg
RUN apk update && apk add --no-cache ffmpeg

# Instalar dependencias necesarias para compilación
RUN apk add --no-cache python3 py3-pip gcc g++ make

# Establecer el directorio de trabajo
WORKDIR /app

# Copiar los archivos de dependencias de Go
COPY go.mod go.sum ./

# Descargar dependencias
RUN go mod download

# Copiar el código fuente
COPY . .

# Construir la aplicación en Go
RUN CGO_ENABLED=1 GOOS=linux go build -o main ./cmd/main.go

# Etapa final
FROM alpine:latest

# Actualizar paquetes e instalar dependencias necesarias
RUN apk update && apk add --no-cache python3 py3-pip gcc g++ make ffmpeg

# Instalar yt-dlp y sus dependencias
RUN python3 -m pip install -U "yt-dlp[default]" --break-system-packages

# Crear directorios para la aplicación
WORKDIR /app
RUN mkdir -p /app/storage /app/db /app/pkg/pyConverter

# Copiar el binario compilado desde la etapa de construcción
COPY --from=builder /app/main .
COPY --from=builder /app/pkg/pyConverter /app/pkg/pyConverter

# Copiar el script de entrypoint y dar permisos de ejecución
COPY ./entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh && ls -l /entrypoint.sh
RUN sed -i 's/\r$//' /entrypoint.sh
RUN chmod +x /app/main

# Exponer el puerto de la aplicación
EXPOSE 3000

# Ejecutar el entrypoint
ENTRYPOINT ["/entrypoint.sh"]