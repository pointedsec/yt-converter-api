import yt_dlp
import sys
import os
import subprocess
import argparse


# Returns the path of the cookie file to use if exists
def get_cookie_file_path(cookies_path: str | None) -> str | None:
    """
    Devuelve la ruta del archivo de cookies si existe.
    Si no existe ninguno, devuelve None.
    """
    if cookies_path and os.path.exists(cookies_path):
        print("Usando cookies %s" % cookies_path)
        return cookies_path

    default_path = "./cookies.txt"
    if os.path.exists(default_path):
        print("Usando cookies %s" % cookies_path)
        return default_path

    return None


# Check if user cookie path exists, if not, check if default cookie path exists.
def check_if_cookies_file_is_present(cookies_path: str | None) -> bool:
    """
    Verifica si existe un archivo de cookies.
    Prioriza el path proporcionado por el usuario, luego el archivo por defecto.
    """
    default_path = "./cookies.txt"

    if cookies_path and os.path.exists(cookies_path):
        return True

    return os.path.exists(default_path)


def convert_m4a_to_mp3(m4a_path: str, mp3_path: str) -> str:
    try:
        # Use ffmpeg to convert M4A to MP3
        subprocess.run(
            ["ffmpeg", "-i", m4a_path, "-codec:a", "libmp3lame", "-q:a", "2", mp3_path],
            check=True,
        )
        return mp3_path
    except subprocess.CalledProcessError as e:
        print(f"Error converting M4A to MP3: {e}")
        return ""


def convert_to_audio(
    output_path: str, youtube_id: str, cookies_path: str = None
) -> str:
    if check_if_cookies_file_is_present(cookies_path):
        ydl_opts = {
            "format": "m4a/bestaudio/best",
            "postprocessors": [
                {
                    "key": "FFmpegExtractAudio",
                    "preferredcodec": "m4a",
                }
            ],
            "outtmpl": output_path + "/%(id)s.%(ext)s",
            "cookiefile": get_cookie_file_path(cookies_path=cookies_path),
        }
    else:
        ydl_opts = {
            "format": "m4a/bestaudio/best",
            "postprocessors": [
                {
                    "key": "FFmpegExtractAudio",
                    "preferredcodec": "m4a",
                }
            ],
            "outtmpl": output_path + "/%(id)s.%(ext)s",
        }

    with yt_dlp.YoutubeDL(ydl_opts) as ydl:
        return output_path + f"/{youtube_id}.m4a"


def get_video_available_resolutions(youtube_url: str, cookies_path: str = None) -> list[str]:
    if check_if_cookies_file_is_present(cookies_path):
        ydl_opts = {
            "quiet": True,
            "cookiefile": get_cookie_file_path(cookies_path=cookies_path),
        }
    else:
        ydl_opts = {
            "quiet": True,
        }

    with yt_dlp.YoutubeDL(ydl_opts) as ydl:
        info = ydl.extract_info(youtube_url, download=False)
        streams = info.get("formats", [])
        resolutions = set()

        for stream in streams:
            if stream.get("vcodec") not in [None, "none"]:
                height = stream.get("height")
                if height:
                    resolutions.add(f"{height}p")

        return sorted(resolutions, key=lambda x: int(x.replace("p", "")))


def convert_to_video(youtube_url: str, resolution: str, output_path: str, cookies_path: str = None) -> str:
    available_resolutions = get_video_available_resolutions(youtube_url)

    if resolution not in available_resolutions:
        print(
            f"❌ Resolution {resolution} not available. Available resolutions: {available_resolutions}"
        )
        return ""

    print(f"⏳ Downloading video {youtube_url} with resolution {resolution}...")

    if check_if_cookies_file_is_present(cookies_path):
        ydl_opts = {
            "format": f"bestvideo[height={resolution[:-1]}]+bestaudio/best",
            "outtmpl": output_path + "/%(id)s-%(resolution)s.%(ext)s",
            "merge_output_format": "mp4",
            "cookiefile": get_cookie_file_path(cookies_path=cookies_path)
        }
    else:
        ydl_opts = {
            "format": f"bestvideo[height={resolution[:-1]}]+bestaudio/best",
            "outtmpl": output_path + "/%(id)s-%(resolution)s.%(ext)s",
            "merge_output_format": "mp4",
        }

    with yt_dlp.YoutubeDL(ydl_opts) as ydl:
        try:
            info_dict = ydl.extract_info(youtube_url, download=True)
            output_filename = ydl.prepare_filename(info_dict)
            return output_filename
        except Exception as e:
            print(f"Error downloading video: {e}")
            return ""


def video_to_youtube_url(video_id: str) -> str:
    return f"https://www.youtube.com/watch?v={video_id}"


def delete_repeated_resolutions(resolutions: list[str]) -> list[str]:
    return list(set(resolutions))


def check_if_path_is_valid_and_absolute(path: str) -> bool:
    return os.path.exists(path) and os.path.isabs(path)


def main():
    parser = argparse.ArgumentParser(
        description="Convertidor de videos de YouTube a audio/video con soporte opcional para cookies"
    )

    parser.add_argument("video_id", help="ID del video de YouTube")
    parser.add_argument(
        "convert_to",
        choices=["audio", "video"],
        help="Formato de conversión: audio o video",
    )
    parser.add_argument(
        "output_path", help="Ruta absoluta donde guardar el archivo de salida"
    )

    parser.add_argument(
        "--cookies", help="Ruta al archivo cookies.txt (opcional)", default=None
    )
    parser.add_argument(
        "--resolution",
        help="Resolución deseada para video (opcional si convert_to=video)",
    )

    args = parser.parse_args()

    # Validaciones básicas
    if not check_if_path_is_valid_and_absolute(args.output_path):
        print("❌ Por favor proporciona una ruta absoluta válida para la salida.")
        sys.exit(1)

    video_url = video_to_youtube_url(args.video_id)

    if args.convert_to == "audio":
        path = convert_to_audio(args.output_path, args.video_id, args.cookies)
        mp3path = convert_m4a_to_mp3(
            path, os.path.join(args.output_path, f"{args.video_id}.mp3")
        )
        os.remove(path)
        print(mp3path)
        sys.exit(0)

    elif args.convert_to == "video":
        if not args.resolution:
            resolutions = get_video_available_resolutions(video_url, args.cookies)
            print("Resoluciones disponibles:", resolutions)
            sys.exit(0)
        path = convert_to_video(
            video_url, args.resolution, args.output_path, args.cookies
        )
        print(path)
        sys.exit(0)


if __name__ == "__main__":
    main()
